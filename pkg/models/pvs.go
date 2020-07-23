package models

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	PVManager *SPVManager
	_         IClusterModel = new(SPV)
)

func init() {
	PVManager = NewK8sModelManager(func() ISyncableManager {
		return &SPVManager{
			SClusterResourceBaseManager: NewClusterResourceBaseManager(
				new(SPV),
				"persistentvolumes_tbl",
				"persistentvolume",
				"persistentvolumes",
				api.ResourceNamePersistentVolume,
				api.KindNamePersistentVolume,
				new(v1.PersistentVolume),
			),
		}
	}).(*SPVManager)
}

type SPVManager struct {
	SClusterResourceBaseManager
}

type SPV struct {
	SClusterResourceBase
}

func (obj *SPV) getPVCShortDesc(pv *v1.PersistentVolume) string {
	var claim string
	if pv.Spec.ClaimRef != nil {
		claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Namespace
	}
	return claim
}

func (obj *SPV) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	pv := k8sObj.(*v1.PersistentVolume)
	detail := api.PersistentVolumeDetailV2{
		ClusterResourceDetail: obj.SClusterResourceBase.GetDetails(cli, base, k8sObj, isList).(api.ClusterResourceDetail),
		Capacity:              pv.Spec.Capacity,
		AccessModes:           pv.Spec.AccessModes,
		ReclaimPolicy:         pv.Spec.PersistentVolumeReclaimPolicy,
		StorageClass:          pv.Spec.StorageClassName,
		Status:                pv.Status.Phase,
		Claim:                 obj.getPVCShortDesc(pv),
		Reason:                pv.Status.Reason,
		Message:               pv.Status.Message,
	}
	if isList {
		return detail
	}
	// todo: add pvc info
	return detail
}
