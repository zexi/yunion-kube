package machines

import (
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type SSystemYKEDriver struct {
	*sBaseDriver
}

func NewSystemYKEDriver() *SSystemYKEDriver {
	return &SSystemYKEDriver{
		sBaseDriver: newBaseDriver(),
	}
}

func (d *SSystemYKEDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeSystem
}

func init() {
	driver := &SSystemYKEDriver{}
	machines.RegisterMachineDriver(driver)
}
