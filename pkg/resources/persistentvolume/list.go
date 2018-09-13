package persistentvolume

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type PersistentVolumeList struct {
	*dataselect.ListMeta
	Items []PersistentVolume
}

// PersistentVolume provides the simplified presentation layer view of Kubernetes Persistent Volume resource.
type PersistentVolume struct {
	api.ObjectMeta
	api.TypeMeta
	Capacity      v1.ResourceList                  `json:"capacity"`
	AccessModes   []v1.PersistentVolumeAccessMode  `json:"accessModes"`
	ReclaimPolicy v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy"`
	StorageClass  string                           `json:"storageClass"`
	Status        v1.PersistentVolumePhase         `json:"status"`
	Claim         string                           `json:"claim"`
	Reason        string                           `json:"reason"`
}

func (man *SPersistentVolumeManager) List(req *common.Request) (common.ListResource, error) {
	return GetPersistentVolumeList(req.GetK8sClient(), req.ToQuery())
}

// GetPersistentVolumeList returns a list of all Persistent Volumes in the cluster.
func GetPersistentVolumeList(client kubernetes.Interface, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeList, error) {
	log.Infof("Getting list persistent volumes")
	channels := &common.ResourceChannels{
		PersistentVolumeList: common.GetPersistentVolumeListChannel(client),
	}

	return GetPersistentVolumeListFromChannels(channels, dsQuery)
}

// GetPersistentVolumeListFromChannels returns a list of all Persistent Volumes in the cluster
// reading required resource list once from the channels.
func GetPersistentVolumeListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeList, error) {
	persistentVolumes := <-channels.PersistentVolumeList.List
	err := <-channels.PersistentVolumeList.Error

	if err != nil {
		return nil, err
	}

	return toPersistentVolumeList(persistentVolumes.Items, dsQuery)
}

func toPersistentVolumeList(persistentVolumes []v1.PersistentVolume, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeList, error) {
	result := &PersistentVolumeList{
		Items:    make([]PersistentVolume, 0),
		ListMeta: dataselect.NewListMeta(),
	}

	err := dataselect.ToResourceList(
		result,
		persistentVolumes,
		dataselect.NewResourceDataCell,
		dsQuery,
	)

	return result, err
}

func (l *PersistentVolumeList) Append(obj interface{}) {
	item := obj.(v1.PersistentVolume)
	l.Items = append(l.Items,
		PersistentVolume{
			ObjectMeta:    api.NewObjectMeta(item.ObjectMeta),
			TypeMeta:      api.NewTypeMeta(api.ResourceKindPersistentVolume),
			Capacity:      item.Spec.Capacity,
			AccessModes:   item.Spec.AccessModes,
			ReclaimPolicy: item.Spec.PersistentVolumeReclaimPolicy,
			StorageClass:  item.Spec.StorageClassName,
			Status:        item.Status.Phase,
			Claim:         getPersistentVolumeClaim(&item),
			Reason:        item.Status.Reason,
		})
}

func (l *PersistentVolumeList) GetResponseData() interface{} {
	return l.Items
}

// getPersistentVolumeClaim returns Persistent Volume claim using "namespace/claim" format.
func getPersistentVolumeClaim(pv *v1.PersistentVolume) string {
	var claim string

	if pv.Spec.ClaimRef != nil {
		claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
	}

	return claim
}
