package persistentvolume

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type PersistentVolumeList struct {
	*common.BaseList
	Items []api.PersistentVolume
}

func (man *SPersistentVolumeManager) List(req *common.Request) (common.ListResource, error) {
	return GetPersistentVolumeList(req.GetIndexer(), req.GetCluster(), req.ToQuery())
}

// GetPersistentVolumeList returns a list of all Persistent Volumes in the cluster.
func GetPersistentVolumeList(client *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeList, error) {
	log.Infof("Getting list persistent volumes")
	channels := &common.ResourceChannels{
		PersistentVolumeList: common.GetPersistentVolumeListChannel(client),
	}

	return GetPersistentVolumeListFromChannels(channels, dsQuery, cluster)
}

// GetPersistentVolumeListFromChannels returns a list of all Persistent Volumes in the cluster
// reading required resource list once from the channels.
func GetPersistentVolumeListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*PersistentVolumeList, error) {
	persistentVolumes := <-channels.PersistentVolumeList.List
	err := <-channels.PersistentVolumeList.Error

	if err != nil {
		return nil, err
	}

	return toPersistentVolumeList(persistentVolumes, dsQuery, cluster)
}

func toPersistentVolumeList(persistentVolumes []*v1.PersistentVolume, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*PersistentVolumeList, error) {
	result := &PersistentVolumeList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]api.PersistentVolume, 0),
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
	item := obj.(*v1.PersistentVolume)
	l.Items = append(l.Items,ToPeristentVolume(item, l.GetCluster()))
}

func ToPeristentVolume(pv *v1.PersistentVolume, cluster api.ICluster) api.PersistentVolume {
	return api.PersistentVolume{
		ObjectMeta:    api.NewObjectMeta(pv.ObjectMeta, cluster),
		TypeMeta:      api.NewTypeMeta(pv.TypeMeta),
		Capacity:      pv.Spec.Capacity,
		AccessModes:   pv.Spec.AccessModes,
		ReclaimPolicy: pv.Spec.PersistentVolumeReclaimPolicy,
		StorageClass:  pv.Spec.StorageClassName,
		Status:        pv.Status.Phase,
		Claim:         getPersistentVolumeClaim(pv),
		Reason:        pv.Status.Reason,
	}
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
