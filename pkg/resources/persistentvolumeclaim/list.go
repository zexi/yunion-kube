package persistentvolumeclaim

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type PersistentVolumeClaimList struct {
	*dataselect.ListMeta
	Items []PersistentVolumeClaim
}

// PersistentVolumeClaim provides the simplified presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaim struct {
	metaV1.ObjectMeta
	metaV1.TypeMeta
	Status       string                          `json:"status"`
	Volume       string                          `json:"volume"`
	Capacity     v1.ResourceList                 `json:"capacity"`
	AccessModes  []v1.PersistentVolumeAccessMode `json:"accessModes"`
	StorageClass *string                         `json:"storageClass"`
	MountedBy    []string                        `json:"mountedBy"`
}

func (man *SPersistentVolumeClaimManager) List(req *common.Request) (common.ListResource, error) {
	query := req.ToQuery()
	if req.Query.Contains("unused") {
		filter := query.FilterQuery
		isUnused := "false"
		if jsonutils.QueryBoolean(req.Query, "unused", false) {
			isUnused = "true"
		}
		filter.Append(dataselect.NewFilterBy(dataselect.PVCUnusedProperty, isUnused))
	}
	return man.ListV2(req.GetK8sClient(), req.GetNamespaceQuery(), query)
}

func (man *SPersistentVolumeClaimManager) ListV2(client kubernetes.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetPersistentVolumeClaimList(client, nsQuery, dsQuery)
}

// GetPersistentVolumeClaimList returns a list of all Persistent Volume Claims in the cluster.
func GetPersistentVolumeClaimList(client kubernetes.Interface, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {
	log.Infof("Getting list persistent volumes claims")
	channels := &common.ResourceChannels{
		PersistentVolumeClaimList: common.GetPersistentVolumeClaimListChannel(client, nsQuery),
		PodList:                   common.GetPodListChannel(client, nsQuery),
	}

	return GetPersistentVolumeClaimListFromChannels(channels, nsQuery, dsQuery)
}

// GetPersistentVolumeClaimListFromChannels returns a list of all Persistent Volume Claims in the cluster
// reading required resource list once from the channels.
func GetPersistentVolumeClaimListFromChannels(channels *common.ResourceChannels, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {

	persistentVolumeClaims := <-channels.PersistentVolumeClaimList.List
	err := <-channels.PersistentVolumeClaimList.Error
	if err != nil {
		return nil, err
	}
	pods := <-channels.PodList.List
	err = <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	pvcs := []PersistentVolumeClaim{}
	for _, pvc := range persistentVolumeClaims.Items {
		pvcs = append(pvcs, toPersistentVolumeClaim(pvc, pods.Items))
	}

	return toPersistentVolumeClaimList(pvcs, dsQuery)
}

func getPvcs(volumes []v1.Volume) []v1.Volume {
	var pvcs []v1.Volume
	for _, vol := range volumes {
		if vol.VolumeSource.PersistentVolumeClaim != nil {
			pvcs = append(pvcs, vol)
		}
	}
	return pvcs
}

func getMountPods(pvcName string, pods []v1.Pod) []v1.Pod {
	ret := []v1.Pod{}
	for _, pod := range pods {
		pvcs := getPvcs(pod.Spec.Volumes)
		for _, pvc := range pvcs {
			if pvc.PersistentVolumeClaim.ClaimName == pvcName {
				ret = append(ret, pod)
			}
		}
	}
	return ret
}

func getMountPodsName(pvcName string, pods []v1.Pod) []string {
	pods = getMountPods(pvcName, pods)
	ret := []string{}
	for _, pod := range pods {
		ret = append(ret, pod.Name)
	}
	return ret
}

func toPersistentVolumeClaim(pvc v1.PersistentVolumeClaim, pods []v1.Pod) PersistentVolumeClaim {
	podsName := getMountPodsName(pvc.Name, pods)
	return PersistentVolumeClaim{
		ObjectMeta:   pvc.ObjectMeta,
		TypeMeta:     pvc.TypeMeta,
		Status:       string(pvc.Status.Phase),
		Volume:       pvc.Spec.VolumeName,
		Capacity:     pvc.Status.Capacity,
		AccessModes:  pvc.Spec.AccessModes,
		StorageClass: pvc.Spec.StorageClassName,
		MountedBy:    podsName,
	}
}

func toPersistentVolumeClaimList(pvcs []PersistentVolumeClaim, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {
	result := &PersistentVolumeClaimList{
		Items:    make([]PersistentVolumeClaim, 0),
		ListMeta: dataselect.NewListMeta(),
	}

	err := dataselect.ToResourceList(
		result,
		pvcs,
		dataselect.NewPVCDataCell,
		dsQuery,
	)

	return result, err
}

func (l *PersistentVolumeClaimList) Append(obj interface{}) {
	item := obj.(PersistentVolumeClaim)
	l.Items = append(l.Items, item)
}

func (l *PersistentVolumeClaimList) GetResponseData() interface{} {
	return l.Items
}
