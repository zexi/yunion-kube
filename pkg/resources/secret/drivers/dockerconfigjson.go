package drivers

import (
	"encoding/base64"
	"fmt"

	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/secret"
)

func init() {
	secret.SecretManager.RegisterDriver(
		v1.SecretTypeDockerConfigJson,
		newDockerConfigJson())
}

type dockerConfigJson struct {}

func newDockerConfigJson() secret.ISecretDriver {
	return new(dockerConfigJson)
}

func (d dockerConfigJson) ValidateCreateData(input *apis.SecretCreateInput) error {
	conf := input.DockerConfigJson
	if conf.User == "" {
		return httperrors.NewInputParameterError("user is empty")
	}
	if conf.Password == "" {
		return httperrors.NewInputParameterError("password is empty")
	}
	return nil
}

func (d dockerConfigJson) toAuth(user, password string) string {
	auth := fmt.Sprintf("%s:%s", user, password)
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (d dockerConfigJson) ToData(input *apis.SecretCreateInput) (map[string]string, error) {
	spec := input.DockerConfigJson
	authInfo := jsonutils.NewDict()
	authInfo.Add(jsonutils.NewString(spec.User), "username")
	authInfo.Add(jsonutils.NewString(spec.Password), "password")
	authInfo.Add(jsonutils.NewString(spec.Email), "email")
	authInfo.Add(jsonutils.NewString(d.toAuth(spec.User, spec.Password)), "auth")
	authRegistry := jsonutils.NewDict()
	authRegistry.Add(authInfo, "auths", spec.Server)
	return map[string]string{
		v1.DockerConfigJsonKey: authRegistry.String(),
	}, nil
}


