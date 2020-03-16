package secret

import (
	v1 "k8s.io/api/core/v1"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/drivers"
)

var (
	SecretManager         *SSecretManager
	RegistrySecretManager *SRegistrySecretManager
)

type SSecretManager struct {
	*resources.SNamespaceResourceManager
	driverManager *drivers.DriverManager
}

type SRegistrySecretManager struct {
	*SSecretManager
}

type SCephCSISecretManager struct {
	*SSecretManager
}

func init() {
	SecretManager = &SSecretManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("secret", "secrets"),
		driverManager: drivers.NewDriverManager(""),
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

type ISecretDriver interface {
	ValidateCreateData(input *api.SecretCreateInput) error
	ToData(input *api.SecretCreateInput) (map[string]string, error)
}

func (m *SSecretManager) RegisterDriver(typ v1.SecretType, driver ISecretDriver) {
	if err := m.driverManager.Register(driver, string(typ)); err != nil {
		panic(errors.Wrapf(err, "secret register driver %s", typ))
	}
}

func (m *SSecretManager) GetDriver(typ v1.SecretType) (ISecretDriver, error) {
	drv, err := m.driverManager.Get(string(typ))
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("secret get %s driver", typ)
		}
		return nil, err
	}
	return drv.(ISecretDriver), nil
}
