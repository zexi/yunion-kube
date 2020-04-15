package k8smodels

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	PVManager *SPVManager
)

func init() {
	PVManager = &SPVManager{
		SK8SClusterResourceBaseManager: model.NewK8SClusterResourceBaseManager(
			new(SPV), "persistentvolume", "persistentvolumes"),
	}
	PVManager.SetVirtualObject(PVManager)
}

type SPVManager struct {
	model.SK8SClusterResourceBaseManager
}

type SPV struct {
	model.SK8SClusterResourceBase
}

func (m *SPVManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNamePersistentVolume,
		Object:       new(v1.PersistentVolume),
		KindName:     apis.KindNamePersistentVolume,
	}
}

func (m *SPVManager) GetRawPVs(cluster model.ICluster) ([]*v1.PersistentVolume, error) {
	return cluster.GetHandler().GetIndexer().PVLister().List(labels.Everything())
}

func (m *SPVManager) GetAPIPVs(cluster model.ICluster, pvs []*v1.PersistentVolume) ([]*apis.PersistentVolume, error) {
	ret := make([]*apis.PersistentVolume, 0)
	err := ConvertRawToAPIObjects(m, cluster, pvs, &ret)
	return ret, err
}

func (obj *SPV) GetRawPV() *v1.PersistentVolume {
	return obj.GetK8SObject().(*v1.PersistentVolume)
}

func (obj *SPV) getPVCShortDesc() string {
	var claim string
	pv := obj.GetRawPV()
	if pv.Spec.ClaimRef != nil {
		claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Namespace
	}
	return claim
}

func (obj *SPV) GetPVC() (*SPVC, error) {
	pvcRef := obj.GetRawPV().Spec.ClaimRef
	if pvcRef == nil {
		return nil, nil
	}
	pvcObj, err := model.NewK8SModelObjectByRef(PVCManager, obj.GetCluster(), pvcRef)
	if err != nil {
		return nil, err
	}
	return pvcObj.(*SPVC), nil
}

func (obj *SPV) GetAPIObject() (*apis.PersistentVolume, error) {
	pv := obj.GetRawPV()
	return &apis.PersistentVolume{
		ObjectMeta:    obj.GetObjectMeta(),
		TypeMeta:      obj.GetTypeMeta(),
		Capacity:      pv.Spec.Capacity,
		AccessModes:   pv.Spec.AccessModes,
		ReclaimPolicy: pv.Spec.PersistentVolumeReclaimPolicy,
		StorageClass:  pv.Spec.StorageClassName,
		Status:        pv.Status.Phase,
		Claim:         obj.getPVCShortDesc(),
		Reason:        pv.Status.Reason,
		Message:       pv.Status.Message,
	}, nil
}

func (obj *SPV) GetAPIDetailObject() (*apis.PersistentVolumeDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	pvc, err := obj.GetPVC()
	if err != nil {
		return nil, err
	}
	var apiPvc *apis.PersistentVolumeClaim
	if pvc != nil {
		apiPvc, err = pvc.GetAPIObject()
		if err != nil {
			return nil, err
		}
	}
	return &apis.PersistentVolumeDetail{
		PersistentVolume:       *apiObj,
		PersistentVolumeSource: obj.GetRawPV().Spec.PersistentVolumeSource,
		PersistentVolumeClaim:  apiPvc,
	}, nil
}
