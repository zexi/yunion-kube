package model

var globalModelManagers map[K8SResourceInfo]IK8SModelManager

func RegisterModelManager(man IK8SModelManager) {
	if globalModelManagers == nil {
		globalModelManagers = make(map[K8SResourceInfo]IK8SModelManager)
	}
	globalModelManagers[man.GetK8SResourceInfo()] = man
}

func GetK8SModelManagerByKind(kindName string) IK8SModelManager {
	for rsInfo, man := range globalModelManagers {
		if rsInfo.KindName == kindName {
			return man
		}
	}
	return nil
}
