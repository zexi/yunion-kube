package secret

import (
	"encoding/base64"
	"fmt"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SSecretManager) ValidateCreateData(req *common.Request) error {
	_, err := req.Data.GetMap("data")
	if err != nil {
		return httperrors.NewInputParameterError("Not found data")
	}
	return man.SNamespaceResourceManager.ValidateCreateData(req)
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

	return common.ValidateK8sResourceCreateData(req, apis.ResourceKindSecret, true)
}

func (man *SSecretManager) Create(req *common.Request) (interface{}, error) {
	dataMap, _ := req.Data.GetMap("data")
	name, _ := req.Data.GetString("name")
	ns := req.GetDefaultNamespace()
	var kind v1.SecretType
	kindStr, _ := req.Data.GetString("secretType")
	if kindStr == "" {
		kind = v1.SecretTypeOpaque
	} else {
		kind = v1.SecretType(kindStr)
	}
	data := make(map[string][]byte)
	for key, obj := range dataMap {
		content, err := obj.GetString()
		if err != nil {
			return nil, err
		}
		data[key] = []byte(content)
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: data,
		Type: kind,
	}
	obj, err := req.GetK8sClient().CoreV1().Secrets(ns).Create(secret)
	return obj, err
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
