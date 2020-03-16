package drivers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/storageclass"
)

const (
	CSIStorageK8SIO = "csi.storage.k8s.io"
)

func init() {
	storageclass.StorageClassManager.RegisterDriver(
		apis.StorageClassProvisionerCephCSIRBD,
		newCephCSIRBD(),
	)
}

func GetCSIParamsKey(suffix string) string {
	return CSIStorageK8SIO + "/" + suffix
}

type CephCSIRBD struct {}

func newCephCSIRBD() storageclass.IStorageClassDriver {
	return new(CephCSIRBD)
}

func (drv *CephCSIRBD) ValidateCreateData(req *common.Request, data *apis.StorageClassCreateInput) error {
	input := data.CephCSIRBD
	if input == nil {
		return httperrors.NewInputParameterError("cephCSIRBD config is empty")
	}
	secretName := input.SecretName
	if secretName == "" {
		return httperrors.NewNotEmptyError("secretName is empty")
	}
	secretNamespace := input.SecretNamespace
	if secretNamespace == "" {
		return httperrors.NewNotEmptyError("secretNamespace is empty")
	}
	cli := req.GetK8sClient()
	if obj, err := cli.CoreV1().Secrets(secretNamespace).Get(secretName, metav1.GetOptions{}); err != nil {
		return err
	} else if obj.Type != apis.SecretTypeCephCSI {
		return httperrors.NewInputParameterError("%s/%s secret type is not %s", secretNamespace, secretName, apis.SecretTypeCephCSI)
	}
	if input.Pool == "" {
		return httperrors.NewInputParameterError("pool is empty")
	} else {
		// check pool
	}
	if input.ClusterId == "" {
		return httperrors.NewInputParameterError("clusterId is empty")
	} else {
		// check clusterId
	}
	if input.CSIFsType == "" {
		return httperrors.NewInputParameterError("csiFsType is empty")
	} else {
		// check csiFSType
	}
	if input.ImageFeatures != "layering" {
		return httperrors.NewInputParameterError("imageFeatures only support 'layering' currently")
	}
	return nil
}

func (drv *CephCSIRBD) ConnectionTest(input *apis.StorageClassCreateInput) (interface{}, error) {
	return nil, nil
}

func (drv *CephCSIRBD) ToStorageClassParams(input *apis.StorageClassCreateInput) (map[string]string, error) {
	config := input.CephCSIRBD
	params := map[string]string{
		"clusterID": config.ClusterId,
		"pool": config.Pool,
		"imageFeatures": config.ImageFeatures,
		GetCSIParamsKey("provisioner-secret-name"): config.SecretName, // config.CSIProvisionerSecretName,
		GetCSIParamsKey("provisioner-secret-namespace"): config.SecretNamespace, // config.CSIProvisionerSecretNamespace,
		GetCSIParamsKey("controller-expand-secret-name"): config.SecretName, // config.CSIControllerExpandSecretName,
		GetCSIParamsKey("controller-expand-secret-namespace"): config.SecretNamespace, // config.CSIControllerExpandSecretNamespace,
		GetCSIParamsKey("node-stage-secret-name"): config.SecretName, // config.CSINodeStageSecretName,
		GetCSIParamsKey("node-stage-secret-namespace"): config.SecretNamespace, // config.CSINodeStageSecretNamespace,
		GetCSIParamsKey("fstype"): config.CSIFsType,
	}
	return params, nil
}
