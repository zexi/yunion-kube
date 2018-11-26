package secret

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var (
	SecretManager         *SSecretManager
	RegistrySecretManager *SRegistrySecretManager
)

type SSecretManager struct {
	*resources.SNamespaceResourceManager
}

type SRegistrySecretManager struct {
	*SSecretManager
}

func init() {
	SecretManager = &SSecretManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("secret", "secrets"),
	}
	RegistrySecretManager = &SRegistrySecretManager{
		SSecretManager: &SSecretManager{
			SNamespaceResourceManager: resources.NewNamespaceResourceManager("registrysecret", "registrysecrets"),
		},
	}
}
