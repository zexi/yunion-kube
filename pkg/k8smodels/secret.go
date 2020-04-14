package k8smodels

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	SecretManager *SSecretManager
	_             model.IPodOwnerModel = &SSecret{}
)

type SSecretManager struct {
	model.SK8SNamespaceResourceBaseManager
	driverManager *drivers.DriverManager
}

func init() {
	SecretManager = &SSecretManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SSecret{},
			"secret",
			"secrets"),
		driverManager: drivers.NewDriverManager(""),
	}
	SecretManager.SetVirtualObject(SecretManager)
}

type ISecretDriver interface {
	ValidateCreateData(input *apis.SecretCreateInput) error
	ToData(input *apis.SecretCreateInput) (map[string]string, error)
}

type SSecret struct {
	model.SK8SNamespaceResourceBase
}

func (m SSecretManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameSecret,
		Object:       &v1.Secret{},
		KindName:     apis.KindNameSecret,
	}
}

func (m SSecretManager) GetDriver(typ v1.SecretType) (ISecretDriver, error) {
	drv, err := m.driverManager.Get(string(typ))
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("secret get %s driver", typ)
		}
		return nil, err
	}
	return drv.(ISecretDriver), nil
}

func (m *SSecretManager) RegisterDriver(typ v1.SecretType, driver ISecretDriver) {
	if err := m.driverManager.Register(driver, string(typ)); err != nil {
		panic(errors.Wrapf(err, "secret register driver %s", typ))
	}
}

func (m SSecretManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input apis.SecretCreateInput) (runtime.Object, error) {
	if input.Type == "" {
		return nil, httperrors.NewNotEmptyError("type is empty")
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	data, err := drv.ToData(&input)
	if err != nil {
		return nil, err
	}
	dataBytes := make(map[string][]byte)
	for k, v := range data {
		dataBytes[k] = []byte(v)
	}
	return &v1.Secret{
		ObjectMeta: input.ToObjectMeta(),
		Type:       input.Type,
		Data:       dataBytes,
	}, nil
}

func (m SSecretManager) ValidateCreateData(
	ctx *model.RequestContext,
	query *jsonutils.JSONDict, input *apis.SecretCreateInput) (*apis.SecretCreateInput, error) {
	if _, err := m.SK8SNamespaceResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sNamespaceResourceCreateInput); err != nil {
		return nil, err
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	return input, drv.ValidateCreateData(input)
}

func (m SSecretManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *apis.ListInputSecret) (model.IQuery, error) {
	q, err := m.SK8SNamespaceResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SNamespaceBase)
	if err != nil {
		return q, err
	}
	if query.Type != "" {
		q.AddFilter(func(obj model.IK8SModel) (bool, error) {
			secret := obj.(*SSecret).GetRawSecret()
			return string(secret.Type) == query.Type, nil
		})
	}
	return q, nil
}

func (m SSecretManager) GetRawSecrets(cluster model.ICluster, ns string) ([]*v1.Secret, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.SecretLister().Secrets(ns).List(labels.Everything())
}

func (m SSecretManager) GetAllRawSecrets(cluster model.ICluster) ([]*v1.Secret, error) {
	return m.GetRawSecrets(cluster, v1.NamespaceAll)
}

func (m *SSecretManager) GetAPISecrets(cluster model.ICluster, ss []*v1.Secret) ([]*apis.Secret, error) {
	ret := make([]*apis.Secret, 0)
	err := ConvertRawToAPIObjects(m, cluster, ss, &ret)
	return ret, err
}

func (obj *SSecret) GetRawSecret() *v1.Secret {
	return obj.GetK8SObject().(*v1.Secret)
}

func (obj *SSecret) GetAPIObject() (*apis.Secret, error) {
	rs := obj.GetRawSecret()
	return &apis.Secret{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Type:       rs.Type,
	}, nil
}

func (obj *SSecret) GetAPIDetailObject() (*apis.SecretDetail, error) {
	rs := obj.GetRawSecret()
	secret, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	return &apis.SecretDetail{
		Secret: *secret,
		Data:   rs.Data,
	}, nil
}

func (obj *SSecret) GetRawPods() ([]*v1.Pod, error) {
	secName := obj.GetName()
	ns := obj.GetNamespace()
	rawPods, err := PodManager.GetRawPods(obj.GetCluster(), ns)
	if err != nil {
		return nil, err
	}
	mountPods := make([]*v1.Pod, 0)
	markMap := make(map[string]bool, 0)
	for _, pod := range rawPods {
		cfgs := GetPodSecretVolumes(pod)
		for _, cfg := range cfgs {
			if cfg.Secret.SecretName == secName {
				if _, ok := markMap[pod.GetName()]; !ok {
					mountPods = append(mountPods, pod)
					markMap[pod.GetName()] = true
				}
			}
		}
	}
	return mountPods, err
}
