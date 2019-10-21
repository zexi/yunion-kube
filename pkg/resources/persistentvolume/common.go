package persistentvolume

import (
	"strings"

	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// GetStorageClassPersistentVolumes gets persistentvolumes that are associated with this storageclass.
func GetStorageClassPersistentVolumes(indexer *client.CacheFactory, cluster api.ICluster, storageClassName string,
	dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeList, error) {

	storageClass, err := indexer.StorageClassLister().Get(storageClassName)

	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PersistentVolumeList: common.GetPersistentVolumeListChannel(
			indexer),
	}

	persistentVolumeList := <-channels.PersistentVolumeList.List

	err = <-channels.PersistentVolumeList.Error
	if err != nil {
		return nil, err
	}

	storagePersistentVolumes := make([]*v1.PersistentVolume, 0)
	for _, pv := range persistentVolumeList {
		if strings.Compare(pv.Spec.StorageClassName, storageClass.Name) == 0 {
			storagePersistentVolumes = append(storagePersistentVolumes, pv)
		}
	}

	log.Infof("Found %d persistentvolumes related to %s storageclass",
		len(storagePersistentVolumes), storageClassName)

	return toPersistentVolumeList(storagePersistentVolumes, dsQuery, cluster)
}
