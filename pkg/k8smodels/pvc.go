package k8smodels

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	PVCManager *SPVCManager
)

func init() {
	PVCManager = &SPVCManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SPVC),
			"persistentvolumeclaim",
			"persistentvolumeclaims"),
	}
	PVCManager.SetVirtualObject(PVCManager)
}

type SPVCManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SPVC struct {
	model.SK8SNamespaceResourceBase
}

func (m *SPVCManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNamePersistentVolumeClaim,
		Object:       &v1.PersistentVolumeClaim{},
	}
}

func (m *SPVCManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *apis.PersistentVolumeClaimListInput) (model.IQuery, error) {
	q, err := m.SK8SNamespaceResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SNamespaceBase)
	if err != nil {
		return q, err
	}
	if query.Unused != nil {
		unused := *query.Unused
		q.AddFilter(func(obj model.IK8SModel) bool {
			pvc := obj.(*SPVC)
			mntPods, err := pvc.getMountRawPods()
			if err != nil {
				panic(err)
			}
			if unused {
				return len(mntPods) == 0
			}
			return len(mntPods) > 0
		})
	}
	return q, nil
}

func (m *SPVCManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input apis.PersistentVolumeClaimCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	size := input.Size
	storageSize, err := resource.ParseQuantity(size)
	if err != nil {
		return nil, err
	}
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

func (obj *SPVC) GetRawPVC() *v1.PersistentVolumeClaim {
	return obj.GetK8SObject().(*v1.PersistentVolumeClaim)
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

func (obj *SPVC) getMountRawPods() ([]*v1.Pod, error) {
	pods, err := PodManager.GetRawPods(obj.GetCluster(), obj.GetNamespace())
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

func (obj *SPVC) GetMountPods() ([]*apis.Pod, error) {
	rawPods, err := obj.getMountRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), rawPods)
}

func (obj *SPVC) GetMountPodNames() ([]string, error) {
	pods, err := obj.getMountRawPods()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0)
	for _, p := range pods {
		names = append(names, p.GetName())
	}
	return names, nil
}

func (obj *SPVC) GetAPIObject() (*apis.PersistentVolumeClaim, error) {
	pvc := obj.GetRawPVC()
	podNames, err := obj.GetMountPodNames()
	if err != nil {
		return nil, err
	}
	return &apis.PersistentVolumeClaim{
		ObjectMeta:   obj.GetObjectMeta(),
		TypeMeta:     obj.GetTypeMeta(),
		Status:       string(pvc.Status.Phase),
		Volume:       pvc.Spec.VolumeName,
		Capacity:     pvc.Status.Capacity,
		AccessModes:  pvc.Spec.AccessModes,
		StorageClass: pvc.Spec.StorageClassName,
		MountedBy:    podNames,
	}, nil
}

func (obj *SPVC) GetAPIDetailObject() (*apis.PersistentVolumeClaimDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	pods, err := obj.GetMountPods()
	if err != nil {
		return nil, err
	}
	return &apis.PersistentVolumeClaimDetail{
		PersistentVolumeClaim: *apiObj,
		Pods:                  pods,
	}, nil
}

func (m *SPVCManager) GetRawPVCs(cluster model.ICluster, ns string) ([]*v1.PersistentVolumeClaim, error) {
	return cluster.GetHandler().GetIndexer().PVCLister().PersistentVolumeClaims(ns).List(labels.Everything())
}

func (m *SPVCManager) GetPodAPIPVCs(cluster model.ICluster, pod *v1.Pod) ([]*apis.PersistentVolumeClaim, error) {
	claimNames := make([]string, 0)
	if pod.Spec.Volumes != nil && len(pod.Spec.Volumes) > 0 {
		for _, v := range pod.Spec.Volumes {
			pvc := v.PersistentVolumeClaim
			if pvc != nil {
				claimNames = append(claimNames, pvc.ClaimName)
			}
		}
	}
	if len(claimNames) == 0 {
		return nil, nil
	}
	allPvcs, err := m.GetRawPVCs(cluster, pod.GetNamespace())
	if err != nil {
		return nil, err
	}
	pvcs := make([]*v1.PersistentVolumeClaim, 0)
	for _, pvc := range allPvcs {
		for _, claimName := range claimNames {
			if strings.Compare(claimName, pvc.Name) == 0 {
				pvcs = append(pvcs, pvc)
				break
			}
		}
	}
	return m.GetAPIPVCs(cluster, pvcs)
}

func (m *SPVCManager) GetAPIPVCs(cluster model.ICluster, pvcs []*v1.PersistentVolumeClaim) ([]*apis.PersistentVolumeClaim, error) {
	ret := make([]*apis.PersistentVolumeClaim, 0)
	if err := ConvertRawToAPIObjects(m, cluster, pvcs, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
