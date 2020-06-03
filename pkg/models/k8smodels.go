package models

import (
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

var globalK8sModelManagers map[K8SResourceInfo]IClusterModelManager

func RegisterK8sModelManager(man IClusterModelManager) {
	if globalK8sModelManagers == nil {
		globalK8sModelManagers = make(map[K8SResourceInfo]IClusterModelManager)
	}
	globalK8sModelManagers[man.GetK8SResourceInfo()] = man
}

// GetK8sResourceManagerByKind used by bidirect sync
func GetK8sResourceManagerByKind(kindName string) manager.IK8sResourceManager {
	for rsInfo, man := range globalK8sModelManagers {
		if rsInfo.KindName == kindName {
			return man.(manager.IK8sResourceManager)
		}
	}
	return nil
}
