package model

import (
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/yunion-kube/pkg/apis"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
)

type K8SResourceInfo struct {
	ResourceName string
	Object       runtime.Object
}

type IK8SModelManager interface {
	lockman.ILockedClass
	object.IObject

	Factory() *SK8SObjectFactory
	KeywordPlural() string
	GetK8SResourceInfo() K8SResourceInfo
	GetQuery(cluster ICluster) IQuery
}

type IK8SModel interface {
	lockman.ILockedObject
	object.IObject

	GetName() string
	KeywordPlural() string

	GetModelManager() IK8SModelManager
	SetModelManager(manager IK8SModelManager, model IK8SModel) IK8SModel

	GetCluster() ICluster
	SetCluster(cluster ICluster) IK8SModel

	SetK8SObject(runtime.Object) IK8SModel
	GetK8SObject() runtime.Object

	GetObjectMeta() apis.ObjectMeta
	GetTypeMeta() apis.TypeMeta
}
