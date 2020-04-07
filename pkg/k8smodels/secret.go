package k8smodels

import (
	v1 "k8s.io/api/core/v1"
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
	_             model.IK8SModel = &SSecret{}
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

func (obj *SSecret) GetAPIDetailsObject() (*apis.SecretDetail, error) {
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
