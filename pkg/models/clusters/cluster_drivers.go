package clusters

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/types"
)

type IClusterDriver interface {
	GetProvider() types.ProviderType

	// GetKubeconfig get current cluster kubeconfig
	GetKubeconfig(cluster *SCluster) (string, error)

	ValidateCreateData(userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) error
	ValidateDeleteCondition() error

	// CreateClusterResource create cluster resource to global k8s cluster
	CreateClusterResource(man *SClusterManager, data *ClusterCreateResource) error
	// ValidateAddMachine validate create machine resource
	ValidateAddMachine(man *SClusterManager, machine *types.Machine) error
	// GetAddonsManifest return addons yaml manifest to be applied to cluster
	GetAddonsManifest(cluster *SCluster) (string, error)
}

var clusterDrivers map[types.ProviderType]IClusterDriver

func init() {
	clusterDrivers = make(map[types.ProviderType]IClusterDriver)
}

func RegisterClusterDriver(driver IClusterDriver) {
	clusterDrivers[driver.GetProvider()] = driver
}

func GetDriver(provider types.ProviderType) IClusterDriver {
	driver, ok := clusterDrivers[provider]
	if ok {
		return driver
	}
	log.Fatalf("Unsupported cluster provider: %s", provider)
	return nil
}
