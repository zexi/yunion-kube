package storageclass

import (
	storage "k8s.io/api/storage/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolume"
)

func (man *SStorageClassManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetStorageClass(req.GetIndexer(), req.GetCluster(), id)
}

// GetStorageClass returns storage class object.
func GetStorageClass(indexer *client.CacheFactory, cluster api.ICluster, name string) (*api.StorageClassDetail, error) {
	log.Infof("Getting details of %s storage class", name)

	storage, err := indexer.StorageClassLister().Get(name)
	if err != nil {
		return nil, err
	}

	persistentVolumeList, err := persistentvolume.GetStorageClassPersistentVolumes(indexer, cluster,
		storage.Name, dataselect.DefaultDataSelect())

	storageClass := toStorageClassDetail(storage, persistentVolumeList, cluster)
	return &storageClass, err
}

func toStorageClassDetail(storageClass *storage.StorageClass,
	persistentVolumeList *persistentvolume.PersistentVolumeList, cluster api.ICluster) api.StorageClassDetail {
	return api.StorageClassDetail{
		StorageClass:         ToStorageClass(storageClass, cluster),
		PersistentVolumeList: persistentVolumeList.Items,
	}
}
