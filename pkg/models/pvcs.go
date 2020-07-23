package models

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	PVCManager *SPVCManager
	_          IClusterModel = new(SPVC)
)

func init() {
	PVCManager = NewK8sModelManager(func() ISyncableManager {
		return &SPVCManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SPVC),
				"persistentvolumeclaims_tbl",
				"persistentvolumeclaim",
				"persistentvolumeclaims",
				api.ResourceNamePersistentVolumeClaim,
				api.KindNamePersistentVolumeClaim,
				new(v1.PersistentVolumeClaim),
			),
		}
	}).(*SPVCManager)
}

type SPVCManager struct {
	SNamespaceResourceBaseManager
}

type SPVC struct {
	SNamespaceResourceBase
}

func (m *SPVCManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.PersistentVolumeClaimCreateInput)
	body.Unmarshal(input)
	size := input.Size
	storageSize, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
	objMeta := input.ToObjectMeta()
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: objMeta,
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					"storage": storageSize,
				},
			},
			StorageClassName: &input.StorageClass,
		},
		Status: v1.PersistentVolumeClaimStatus{},
	}
	return pvc, nil
}

func (_ *SPVCManager) GetPVCVolumes(vols []v1.Volume) []v1.Volume {
	var pvcs []v1.Volume
	for _, vol := range vols {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvcs = append(pvcs, vol)
		}
	}
	return pvcs
}

func (obj *SPVC) getMountRawPods(cli *client.ClusterManager, pvc *v1.PersistentVolumeClaim) ([]*v1.Pod, error) {
	pods, err := PodManager.GetRawPodsByObjectNamespace(cli, pvc)
	if err != nil {
		return nil, err
	}
	mPods := make([]*v1.Pod, 0)
	for _, pod := range pods {
		pvcs := PVCManager.GetPVCVolumes(pod.Spec.Volumes)
		for _, pvc := range pvcs {
			if pvc.PersistentVolumeClaim.ClaimName == obj.GetName() {
				mPods = append(mPods, pod)
			}
		}
	}
	return mPods, nil
}

func (obj *SPVC) GetMountPodNames(cli *client.ClusterManager, pvc *v1.PersistentVolumeClaim) ([]string, error) {
	pods, err := obj.getMountRawPods(cli, pvc)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, p := range pods {
		names = append(names, p.GetName())
	}
	return names, nil
}

func (obj *SPVC) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	pvc := k8sObj.(*v1.PersistentVolumeClaim)
	detail := api.PersistentVolumeClaimDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Status:                  string(pvc.Status.Phase),
		Volume:                  pvc.Spec.VolumeName,
		Capacity:                pvc.Status.Capacity,
		AccessModes:             pvc.Spec.AccessModes,
		StorageClass:            pvc.Spec.StorageClassName,
	}
	if isList {
		return detail
	}
	if podNames, err := obj.GetMountPodNames(cli, pvc); err != nil {
		log.Errorf("get mount pods error: %v", err)
	} else {
		detail.MountedBy = podNames
	}
	return detail
}
