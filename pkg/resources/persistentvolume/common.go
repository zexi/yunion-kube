package persistentvolume

import (
	"strings"

	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

// GetStorageClassPersistentVolumes gets persistentvolumes that are associated with this storageclass.
func GetStorageClassPersistentVolumes(client client.Interface, storageClassName string,
	dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeList, error) {

	storageClass, err := client.StorageV1().StorageClasses().Get(storageClassName, metaV1.GetOptions{})

	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PersistentVolumeList: common.GetPersistentVolumeListChannel(
			client),
	}

	persistentVolumeList := <-channels.PersistentVolumeList.List

	err = <-channels.PersistentVolumeList.Error
	if err != nil {
		return nil, err
	}

	storagePersistentVolumes := make([]v1.PersistentVolume, 0)
	for _, pv := range persistentVolumeList.Items {
		if strings.Compare(pv.Spec.StorageClassName, storageClass.Name) == 0 {
			storagePersistentVolumes = append(storagePersistentVolumes, pv)
		}
	}

	log.Infof("Found %d persistentvolumes related to %s storageclass",
		len(storagePersistentVolumes), storageClassName)

	return toPersistentVolumeList(storagePersistentVolumes, dsQuery)
}
