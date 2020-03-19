package secret

import (
	"encoding/base64"
	"fmt"

	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	tapis "yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SSecretManager) ValidateCreateData(req *common.Request) error {
	if err := man.SNamespaceResourceManager.ValidateCreateData(req); err != nil {
		return err
	}
	input := new(apis.SecretCreateInput)
	if err := req.DataUnmarshal(input); err != nil {
		return err
	}
	if input.Type == "" {
		return httperrors.NewNotEmptyError("type is empty")
	}
	drv, err := man.GetDriver(input.Type)
	if err != nil {
		return err
	}
	return drv.ValidateCreateData(input)
}

func (man *SRegistrySecretManager) ValidateCreateData(req *common.Request) error {
	data := req.Data
	user, _ := data.GetString("user")
	if user == "" {
		return httperrors.NewInputParameterError("user is empty")
	}
	passwd, _ := data.GetString("password")
	if passwd == "" {
		return httperrors.NewInputParameterError("password is empty")
	}

	return common.ValidateK8sResourceCreateData(req, tapis.ResourceKindSecret, true)
}

func (man *SSecretManager) Create(req *common.Request) (interface{}, error) {
	input := new(apis.SecretCreateInput)
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	drv, err := man.GetDriver(input.Type)
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
	objMeta, err := common.GetK8sObjectCreateMeta(req.Data)
	if err != nil {
		return nil, err
	}
	ns := req.GetDefaultNamespace()
	secret := &v1.Secret{
		ObjectMeta: *objMeta,
		Data:       dataBytes,
		Type:       input.Type,
	}
	obj, err := req.GetK8sClient().CoreV1().Secrets(ns).Create(secret)
	if err != nil {
		return nil, err
	}
	return getSecretDetail(obj, req.GetCluster()), nil
}

type registrySecretSpec struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Server   string `json:"server"`
	Email    string `json:"email"`
}

func (spec registrySecretSpec) toAuth() string {
	auth := fmt.Sprintf("%s:%s", spec.User, spec.Password)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (man *SRegistrySecretManager) Create(req *common.Request) (interface{}, error) {
	data := req.Data
	spec := registrySecretSpec{}
	err := data.Unmarshal(&spec)
	if err != nil {
		return nil, err
	}
	authInfo := jsonutils.NewDict()
	authInfo.Add(jsonutils.NewString(spec.User), "username")
	authInfo.Add(jsonutils.NewString(spec.Password), "password")
	authInfo.Add(jsonutils.NewString(spec.Email), "email")
	authInfo.Add(jsonutils.NewString(spec.toAuth()), "auth")
	authRegistry := jsonutils.NewDict()
	authRegistry.Add(authInfo, "auths", spec.Server)

	secretData := jsonutils.NewDict()
	secretData.Add(jsonutils.NewString(authRegistry.String()), string(v1.DockerConfigJsonKey))
	data.Add(secretData, "data")
	data.Add(jsonutils.NewString(string(v1.SecretTypeDockerConfigJson)), "secretType")
	return man.SSecretManager.Create(req)
}
