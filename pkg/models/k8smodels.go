package models

import (
	"context"
	"reflect"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
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

func NewK8sModelManager(factoryF func() IClusterModelManager) IClusterModelManager {
	man := factoryF()
	man.SetVirtualObject(man)
	RegisterK8sModelManager(man)
	return man
}

type SK8sOwnedResourceBaseManager struct {
	ownerManager IClusterModelManager
}

type IK8sOwnedResource interface {
	IsOwnedBy(ownerModel IClusterModel) (bool, error)
}

func (m SK8sOwnedResourceBaseManager) newOwnerModel(obj jsonutils.JSONObject) (IK8sOwnedResource, error) {
	model, err := db.NewModelObject(m.ownerManager)
	if err != nil {
		return nil, errors.Wrap(err, "db.NewModelObject")
	}
	if err := obj.Unmarshal(model); err != nil {
		return nil, errors.Wrap(err, "json marshal")
	}
	return model.(IK8sOwnedResource), nil
}

func (m SK8sOwnedResourceBaseManager) CustomizeFilterList(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*db.CustomizeListFilters, error) {
	input := new(api.ListInputOwner)
	if err := query.Unmarshal(input); err != nil {
		return nil, err
	}
	filters := db.NewCustomizeListFilters()
	if !input.ShouldDo() {
		return filters, nil
	}
	man := GetK8sResourceManagerByKind(input.OwnerKind)
	if man == nil {
		return filters, httperrors.NewNotFoundError("Not found owner_kind %s", input.OwnerKind)
	}
	ff := func(obj jsonutils.JSONObject) (bool, error) {
		model, err := m.newOwnerModel(obj)
		if err != nil {
			return false, errors.Wrap(err, "newOwnerModel")
		}
		clusterId, _ := obj.GetString("cluster_id")
		namespaceId, _ := obj.GetString("namespace_id")
		ownerModel, err := FetchClusterResourceByIdOrName(man.(IClusterModelManager), userCred, clusterId, namespaceId, input.OwnerName)
		if err != nil {
			return false, errors.Wrapf(err, "get %s/%s/%s/%s", clusterId, namespaceId, input.OwnerKind, input.OwnerName)
		}
		return model.IsOwnedBy(ownerModel)
	}
	filters.Append(ff)
	return filters, nil
}

type IPodOwnerModel interface {
	IClusterModel

	GetRawPods(cli *client.ClusterManager, obj runtime.Object) ([]*v1.Pod, error)
}

func IsPodOwner(owner IPodOwnerModel, pod *SPod) (bool, error) {
	ownerObj, err := GetK8sObject(owner)
	if err != nil {
		return false, errors.Wrap(err, "get owner k8s object")
	}
	cli, err := owner.GetClusterClient()
	if err != nil {
		return false, errors.Wrap(err, "get cluster client")
	}
	pods, err := owner.GetRawPods(cli, ownerObj)
	if err != nil {
		return false, errors.Wrap(err, "get owner raw pods")
	}
	p, err := GetK8sObject(pod)
	if err != nil {
		return false, errors.Wrap(err, "get k8s pod")
	}
	return IsObjectContains(p.(*v1.Pod), pods), nil
}

func IsObjectContains(obj metav1.Object, objs interface{}) bool {
	objsV := reflect.ValueOf(objs)
	for i := 0; i < objsV.Len(); i++ {
		ov := objsV.Index(i).Interface().(metav1.Object)
		if obj.GetUID() == ov.GetUID() {
			return true
		}
	}
	return false
}
