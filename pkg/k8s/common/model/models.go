package model

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/client"
)

type IModelManager interface {
	lockman.ILockedClass
	object.IObject

	GetK8sResourceInfo() K8sResourceInfo
	GetOwnerModel(userCred mcclient.TokenCredential, manager IModelManager, cluster ICluster, namespace string, nameOrId string) (IOwnerModel, error)
}

var (
	GetK8sModelManagerByKind func(kindName string) IModelManager
)

func GetK8SModelObject(cm *client.ClusterManager, managerKind, id string) (*jsonutils.JSONDict, error) {
	cli := cm.GetHandler()
	man := GetK8sModelManagerByKind(managerKind)
	resInfo := man.GetK8sResourceInfo()
	obj, err := cli.Get(resInfo.ResourceName, "", id)
	if err != nil {
		return nil, errors.Wrap(err, "client get resouce object")
	}
	iK8sNode, err := NewK8SModelObject(man.(IK8sModelManager), cm, obj)
	if err != nil {
		return nil, errors.Wrap(err, "new k8s models object")
	}
	return GetObject(iK8sNode)
}
