package model

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"

	"yunion.io/x/yunion-kube/pkg/api"
)

type K8SResourceInfo struct {
	ResourceName string
	KindName     string
	Object       runtime.Object
}

type IK8SModelManager interface {
	lockman.ILockedClass
	object.IObject

	Factory() *SK8SObjectFactory
	KeywordPlural() string
	GetK8SResourceInfo() K8SResourceInfo
	GetQuery(cluster ICluster) IQuery
	GetOrderFields() OrderFields
	RegisterOrderFields(fields ...IOrderField)
}

type IK8SModel interface {
	lockman.ILockedObject
	object.IObject

	GetName() string
	GetNamespace() string
	KeywordPlural() string

	GetModelManager() IK8SModelManager
	SetModelManager(manager IK8SModelManager, model IK8SModel) IK8SModel

	GetCluster() ICluster
	SetCluster(cluster ICluster) IK8SModel

	SetK8SObject(runtime.Object) IK8SModel
	GetK8SObject() runtime.Object

	GetObjectMeta() api.ObjectMeta
	GetTypeMeta() api.TypeMeta
}

type IPodOwnerModel interface {
	IK8SModel

	GetRawPods() ([]*v1.Pod, error)
}

type IServiceOwnerModel interface {
	IK8SModel

	GetRawServices() ([]*v1.Service, error)
}
