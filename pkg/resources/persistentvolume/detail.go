package persistentvolume

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SPersistentVolumeManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetPersistentVolumeDetail(req.GetIndexer(), req.GetCluster(), id)
}

// GetPersistentVolumeDetail returns detailed information about a persistent volume
func GetPersistentVolumeDetail(indexer *client.CacheFactory, cluster api.ICluster, name string) (*api.PersistentVolumeDetail, error) {
	log.Infof("Getting details of %s persistent volume", name)

	rawPersistentVolume, err := indexer.PVLister().Get(name)
	if err != nil {
		return nil, err
	}

	return getPersistentVolumeDetail(rawPersistentVolume, cluster), nil
}

func getPersistentVolumeDetail(persistentVolume *v1.PersistentVolume, cluster api.ICluster) *api.PersistentVolumeDetail {
	return &api.PersistentVolumeDetail{
		PersistentVolume:       ToPeristentVolume(persistentVolume, cluster),
		Message:                persistentVolume.Status.Message,
		PersistentVolumeSource: persistentVolume.Spec.PersistentVolumeSource,
	}
}
