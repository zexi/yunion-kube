package secret

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var SecretManager *SSecretManager

type SSecretManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	SecretManager = &SSecretManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("secret", "secrets"),
	}
}
