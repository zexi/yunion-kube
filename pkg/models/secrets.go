package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/drivers"
)

var (
	secretManager *SSecretManager
	_             IPodOwnerModel = new(SSecret)
)

func GetSecretManager() *SSecretManager {
	if secretManager == nil {
		secretManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SSecretManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					new(SSecret),
					"secrets_tbl",
					"secret",
					"secrets",
					api.ResourceNameDaemonSet,
					api.KindNameSecret,
					new(v1.Secret),
				),
				driverManager: drivers.NewDriverManager(""),
			}
		}).(*SSecretManager)
	}
	return secretManager
}

func init() {
	GetSecretManager()
}

type ISecretDriver interface {
	ValidateCreateData(input *api.SecretCreateInput) error
	ToData(input *api.SecretCreateInput) (map[string]string, error)
}

type SSecretManager struct {
	SNamespaceResourceBaseManager
	driverManager *drivers.DriverManager
}

type SSecret struct {
	SNamespaceResourceBase
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

func (m *SSecretManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.SecretCreateInput) (*api.SecretCreateInput, error) {
	if _, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput); err != nil {
		return input, err
	}
	if input.Type == "" {
		return nil, httperrors.NewNotEmptyError("type is empty")
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	return input, drv.ValidateCreateData(input)
}

func (obj *SSecret) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	rawPods, err := PodManager.GetRawPodsByObjectNamespace(cli, rawObj)
	if err != nil {
		return nil, err
	}
	secName := obj.GetName()
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

func (obj *SSecret) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	rs := k8sObj.(*v1.Secret)
	detail := api.SecretDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Type:                    rs.Type,
	}
	if isList {
		return detail
	}
	detail.Data = rs.Data
	return detail
}

func (m *SSecretManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.SecretCreateInput)
	body.Unmarshal(input)
	if input.Type == "" {
		return nil, httperrors.NewNotEmptyError("type is empty")
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	data, err := drv.ToData(input)
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
