package model

import (
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/yunion-kube/pkg/client"
)

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

func GetK8SModelObject(cm *client.ClusterManager, managerKind, id string) (*jsonutils.JSONDict, error) {
	cli := cm.GetHandler()
	man := GetK8SModelManagerByKind(managerKind)
	resInfo := man.GetK8SResourceInfo()
	obj, err := cli.Get(resInfo.ResourceName, "", id)
	if err != nil {
		return nil, errors.Wrap(err, "client get resouce object")
	}
	iK8sNode, err := NewK8SModelObject(man, cm, obj)
	if err != nil {
		return nil, errors.Wrap(err, "new k8s models object")
	}
	return GetObject(iK8sNode)
}
