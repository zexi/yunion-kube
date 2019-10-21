package persistentvolume

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// PersistentVolumeDetail provides the presentation layer view of Kubernetes Persistent Volume resource.
type PersistentVolumeDetail struct {
	api.ObjectMeta
	api.TypeMeta
	Status                 v1.PersistentVolumePhase         `json:"status"`
	Claim                  string                           `json:"claim"`
	ReclaimPolicy          v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy"`
	AccessModes            []v1.PersistentVolumeAccessMode  `json:"accessModes"`
	StorageClass           string                           `json:"storageClass"`
	Capacity               v1.ResourceList                  `json:"capacity"`
	Message                string                           `json:"message"`
	PersistentVolumeSource v1.PersistentVolumeSource        `json:"persistentVolumeSource"`
	Reason                 string                           `json:"reason"`
}

func (man *SPersistentVolumeManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetPersistentVolumeDetail(req.GetIndexer(), req.GetCluster(), id)
}

// GetPersistentVolumeDetail returns detailed information about a persistent volume
func GetPersistentVolumeDetail(indexer *client.CacheFactory, cluster api.ICluster, name string) (*PersistentVolumeDetail, error) {
	log.Infof("Getting details of %s persistent volume", name)

	rawPersistentVolume, err := indexer.PVLister().Get(name)
	if err != nil {
		return nil, err
	}

	return getPersistentVolumeDetail(rawPersistentVolume, cluster), nil
}

func getPersistentVolumeDetail(persistentVolume *v1.PersistentVolume, cluster api.ICluster) *PersistentVolumeDetail {
	return &PersistentVolumeDetail{
		ObjectMeta:             api.NewObjectMetaV2(persistentVolume.ObjectMeta, cluster),
		TypeMeta:               api.NewTypeMeta(api.ResourceKindPersistentVolume),
		Status:                 persistentVolume.Status.Phase,
		Claim:                  getPersistentVolumeClaim(persistentVolume),
		ReclaimPolicy:          persistentVolume.Spec.PersistentVolumeReclaimPolicy,
		AccessModes:            persistentVolume.Spec.AccessModes,
		StorageClass:           persistentVolume.Spec.StorageClassName,
		Capacity:               persistentVolume.Spec.Capacity,
		Message:                persistentVolume.Status.Message,
		PersistentVolumeSource: persistentVolume.Spec.PersistentVolumeSource,
		Reason:                 persistentVolume.Status.Reason,
	}
}
