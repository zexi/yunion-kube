package secret

import (
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8smodels"
)

func init() {
	k8smodels.SecretManager.RegisterDriver(
		apis.SecretTypeCephCSI,
		newCephCSI(),
	)
}

type cephCSI struct{}

func (c cephCSI) ValidateCreateData(input *apis.SecretCreateInput) error {
	conf := input.CephCSI
	if conf == nil {
		return httperrors.NewInputParameterError("ceph csi config is empty")
	}
	if conf.UserId == "" {
		return httperrors.NewInputParameterError("userId is empty")
	}
	if conf.UserKey == "" {
		return httperrors.NewInputParameterError("userKey is empty")
	}
	return nil
}

func (c cephCSI) ToData(input *apis.SecretCreateInput) (map[string]string, error) {
	conf := input.CephCSI
	ret := map[string]string{
		"userID":  conf.UserId,
		"userKey": conf.UserKey,
	}
	if conf.EncryptionPassphrase != "" {
		ret["encryptionPassphrase"] = conf.EncryptionPassphrase
	}
	return ret, nil
}

func newCephCSI() k8smodels.ISecretDriver {
	return new(cephCSI)
}
