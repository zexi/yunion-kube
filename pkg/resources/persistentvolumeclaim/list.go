package persistentvolumeclaim

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
	"yunion.io/x/yunion-kube/pkg/client"
)

type PersistentVolumeClaimList struct {
	*common.BaseList
	Items []PersistentVolumeClaim
}

// PersistentVolumeClaim provides the simplified presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaim struct {
	api.ObjectMeta
	api.TypeMeta
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
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), query)
}

func (man *SPersistentVolumeClaimManager) ListV2(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetPersistentVolumeClaimList(indexer, cluster, nsQuery, dsQuery)
}

// GetPersistentVolumeClaimList returns a list of all Persistent Volume Claims in the cluster.
func GetPersistentVolumeClaimList(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery,
) (*PersistentVolumeClaimList, error) {
	log.Infof("Getting list persistent volumes claims")
	channels := &common.ResourceChannels{
		PersistentVolumeClaimList: common.GetPersistentVolumeClaimListChannel(indexer, nsQuery),
		PodList:                   common.GetPodListChannel(indexer, nsQuery),
	}

	return GetPersistentVolumeClaimListFromChannels(channels, nsQuery, dsQuery, cluster)
}

// GetPersistentVolumeClaimListFromChannels returns a list of all Persistent Volume Claims in the cluster
// reading required resource list once from the channels.
func GetPersistentVolumeClaimListFromChannels(
	channels *common.ResourceChannels,
	nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery,
	cluster api.ICluster,
) (*PersistentVolumeClaimList, error) {

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
	for _, pvc := range persistentVolumeClaims {
		pvcs = append(pvcs, toPersistentVolumeClaim(pvc, pods, cluster))
	}

	return toPersistentVolumeClaimList(pvcs, dsQuery, cluster)
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

func getMountPods(pvcName string, pods []*v1.Pod) []*v1.Pod {
	ret := []*v1.Pod{}
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

func getMountPodsName(pvcName string, pods []*v1.Pod) []string {
	pods = getMountPods(pvcName, pods)
	ret := []string{}
	for _, pod := range pods {
		ret = append(ret, pod.Name)
	}
	return ret
}

func toPersistentVolumeClaim(pvc *v1.PersistentVolumeClaim, pods []*v1.Pod, cluster api.ICluster) PersistentVolumeClaim {
	podsName := getMountPodsName(pvc.Name, pods)
	return PersistentVolumeClaim{
		ObjectMeta:   api.NewObjectMetaV2(pvc.ObjectMeta, cluster),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindPersistentVolumeClaim),
		Status:       string(pvc.Status.Phase),
		Volume:       pvc.Spec.VolumeName,
		Capacity:     pvc.Status.Capacity,
		AccessModes:  pvc.Spec.AccessModes,
		StorageClass: pvc.Spec.StorageClassName,
		MountedBy:    podsName,
	}
}

func toPersistentVolumeClaimList(pvcs []PersistentVolumeClaim, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*PersistentVolumeClaimList, error) {
	result := &PersistentVolumeClaimList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]PersistentVolumeClaim, 0),
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
