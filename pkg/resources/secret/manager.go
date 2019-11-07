package secret

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
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
	resources.KindManagerMap.Register(api.KindNameSecret, SecretManager)
	RegistrySecretManager = &SRegistrySecretManager{
		SSecretManager: &SSecretManager{
			SNamespaceResourceManager: resources.NewNamespaceResourceManager("registrysecret", "registrysecrets"),
		},
	}
}

func (m *SSecretManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetSecretDetail(cli, cluster, namespace, name)
}
