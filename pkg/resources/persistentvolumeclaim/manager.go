package persistentvolumeclaim

import (
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
}
