package storageclass

import (
	"database/sql"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/ceph"

	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/k8smodels"
)

const (
	CSIStorageK8SIO = "csi.storage.k8s.io"
)

func init() {
	k8smodels.StorageClassManager.RegisterDriver(
		api.StorageClassProvisionerCephCSIRBD,
		newCephCSIRBD(),
	)
}

func GetCSIParamsKey(suffix string) string {
	return CSIStorageK8SIO + "/" + suffix
}

type CephCSIRBD struct{}

func newCephCSIRBD() k8smodels.IStorageClassDriver {
	return new(CephCSIRBD)
}

func (drv *CephCSIRBD) getUserKeyFromSecret(ctx *model.RequestContext, name, namespace string) (string, string, error) {
	cli := ctx.Cluster().GetClientset()
	secret, err := cli.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	} else if secret.Type != api.SecretTypeCephCSI {
		return "", "", httperrors.NewInputParameterError("%s/%s secret type is not %s", namespace, name, api.SecretTypeCephCSI)
	}
	uId := string(secret.Data["userID"])
	key := string(secret.Data["userKey"])
	if err != nil {
		return "", "", httperrors.NewNotAcceptableError("%s/%s user key decode error: %v", namespace, name, err)
	}
	return uId, key, nil
}

type cephConfig struct {
	api.ComponentCephCSIConfigCluster
	User string
	Key  string
}

func (drv *CephCSIRBD) getCephConfig(ctx *model.RequestContext, data *api.StorageClassCreateInput) (*cephConfig, error) {
	input := data.CephCSIRBD
	if input == nil {
		return nil, httperrors.NewInputParameterError("cephCSIRBD config is empty")
	}
	secretName := input.SecretName
	if secretName == "" {
		return nil, httperrors.NewNotEmptyError("secretName is empty")
	}
	secretNamespace := input.SecretNamespace
	if secretNamespace == "" {
		return nil, httperrors.NewNotEmptyError("secretNamespace is empty")
	}

	user, key, err := drv.getUserKeyFromSecret(ctx, secretName, secretNamespace)
	if err != nil {
		return nil, err
	}

	cluster := ctx.Cluster().GetClusterObject().(*models.SCluster)
	// check clusterId
	component, err := cluster.GetComponentByType(api.ClusterComponentCephCSI)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("not found cluster %s component %s", cluster.GetName(), api.ClusterComponentCephCSI)
		}
		return nil, err
	}
	settings, err := component.GetSettings()
	if err != nil {
		return nil, err
	}
	if input.ClusterId == "" {
		return nil, httperrors.NewInputParameterError("clusterId is empty")
	}
	cephConf, err := drv.validateClusterId(input.ClusterId, settings.CephCSI)
	if err != nil {
		return nil, err
	}
	return &cephConfig{
		cephConf,
		user,
		key,
	}, nil
}

func (drv *CephCSIRBD) ValidateCreateData(ctx *model.RequestContext, data *api.StorageClassCreateInput) (*api.StorageClassCreateInput, error) {
	cephConf, err := drv.getCephConfig(ctx, data)
	if err != nil {
		return nil, err
	}

	input := data.CephCSIRBD

	if input.Pool == "" {
		return nil, httperrors.NewInputParameterError("pool is empty")
	}
	if err := drv.validatePool(cephConf.Monitors, cephConf.User, cephConf.Key, input.Pool); err != nil {
		return nil, err
	}

	if input.CSIFsType == "" {
		return nil, httperrors.NewInputParameterError("csiFsType is empty")
	} else {
		if !utils.IsInStringArray(input.CSIFsType, []string{"ext4", "xfs"}) {
			return nil, httperrors.NewInputParameterError("unsupport fsType %s", input.CSIFsType)
		}
	}

	if input.ImageFeatures != "layering" {
		return nil, httperrors.NewInputParameterError("imageFeatures only support 'layering' currently")
	}
	return data, nil
}

func (drv *CephCSIRBD) listPools(monitors []string, user string, key string) ([]string, error) {
	cephCli, err := ceph.NewClient(user, key, monitors...)
	if err != nil {
		return nil, errors.Wrap(err, "new ceph client")
	}
	return cephCli.ListPoolsNoDefault()
}

func (drv *CephCSIRBD) validateClusterId(cId string, conf *api.ComponentSettingCephCSI) (api.ComponentCephCSIConfigCluster, error) {
	for _, c := range conf.Config {
		if c.ClsuterId == cId {
			return c, nil
		}
	}
	return api.ComponentCephCSIConfigCluster{}, httperrors.NewNotFoundError("Not found clusterId %s in component config", cId)
}

func (drv *CephCSIRBD) validatePool(monitors []string, user string, key string, pool string) error {
	pools, err := drv.listPools(monitors, user, key)
	if err != nil {
		return err
	}
	if !utils.IsInStringArray(pool, pools) {
		return httperrors.NewNotFoundError("not found pool %s in %v", pool, monitors)
	}
	return nil
}

func (drv *CephCSIRBD) ConnectionTest(ctx *model.RequestContext, data *api.StorageClassCreateInput) (*api.StorageClassTestResult, error) {
	cephConf, err := drv.getCephConfig(ctx, data)
	if err != nil {
		return nil, err
	}
	pools, err := drv.listPools(cephConf.Monitors, cephConf.User, cephConf.Key)
	if err != nil {
		return nil, err
	}
	ret := new(api.StorageClassTestResult)
	ret.CephCSIRBD = &api.StorageClassTestResultCephCSIRBD{Pools: pools}
	return ret, nil
}

func (drv *CephCSIRBD) ToStorageClassParams(input *api.StorageClassCreateInput) (map[string]string, error) {
	config := input.CephCSIRBD
	params := map[string]string{
		"clusterID":     config.ClusterId,
		"pool":          config.Pool,
		"imageFeatures": config.ImageFeatures,
		GetCSIParamsKey("provisioner-secret-name"):            config.SecretName,      // config.CSIProvisionerSecretName,
		GetCSIParamsKey("provisioner-secret-namespace"):       config.SecretNamespace, // config.CSIProvisionerSecretNamespace,
		GetCSIParamsKey("controller-expand-secret-name"):      config.SecretName,      // config.CSIControllerExpandSecretName,
		GetCSIParamsKey("controller-expand-secret-namespace"): config.SecretNamespace, // config.CSIControllerExpandSecretNamespace,
		GetCSIParamsKey("node-stage-secret-name"):             config.SecretName,      // config.CSINodeStageSecretName,
		GetCSIParamsKey("node-stage-secret-namespace"):        config.SecretNamespace, // config.CSINodeStageSecretNamespace,
		GetCSIParamsKey("fstype"):                             config.CSIFsType,
	}
	return params, nil
}
