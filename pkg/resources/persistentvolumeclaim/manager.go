package persistentvolumeclaim

import (
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
)

var PersistentVolumeClaimManager *SPersistentVolumeClaimManager

type SPersistentVolumeClaimManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	PersistentVolumeClaimManager = &SPersistentVolumeClaimManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("persistentvolumeclaim", "persistentvolumeclaims"),
	}
	resources.KindManagerMap.Register(apis.KindNamePersistentVolumeClaim, PersistentVolumeClaimManager)
}

func (m *SPersistentVolumeClaimManager) GetDetails(cli *client.CacheFactory, cluster apis.ICluster, namespace, name string) (interface{}, error) {
	return GetPersistentVolumeClaimDetail(cli, cluster, namespace, name)
}
