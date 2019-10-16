package storageclass

import (
	storage "k8s.io/api/storage/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/persistentvolume"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// StorageClass is a representation of a kubernetes StorageClass object.
type StorageClass struct {
	api.ObjectMeta
	api.TypeMeta

	// provisioner is the driver expected to handle this StorageClass.
	// This is an optionally-prefixed name, like a label key.
	// For example: "kubernetes.io/gce-pd" or "kubernetes.io/aws-ebs".
	// This value may not be empty.
	Provisioner string `json:"provisioner"`

	// parameters holds parameters for the provisioner.
	// These values are opaque to the  system and are passed directly
	// to the provisioner.  The only validation done on keys is that they are
	// not empty.  The maximum number of parameters is
	// 512, with a cumulative max size of 256K
	// +optional
	Parameters map[string]string `json:"parameters"`
}

// StorageClassDetail provides the presentation layer view of Kubernetes StorageClass resource,
// It is StorageClassDetail plus PersistentVolumes associated with StorageClass.
type StorageClassDetail struct {
	api.ObjectMeta
	api.TypeMeta
	Provisioner          string                                `json:"provisioner"`
	Parameters           map[string]string                     `json:"parameters"`
	PersistentVolumeList persistentvolume.PersistentVolumeList `json:"persistentVolumeList"`
}

func (man *SStorageClassManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetStorageClass(req.GetIndexer(), req.GetCluster(), id)
}

// GetStorageClass returns storage class object.
func GetStorageClass(indexer *client.CacheFactory, cluster api.ICluster, name string) (*StorageClassDetail, error) {
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
	persistentVolumeList *persistentvolume.PersistentVolumeList, cluster api.ICluster) StorageClassDetail {
	return StorageClassDetail{
		ObjectMeta:           api.NewObjectMetaV2(storageClass.ObjectMeta, cluster),
		TypeMeta:             api.NewTypeMeta(api.ResourceKindStorageClass),
		Provisioner:          storageClass.Provisioner,
		Parameters:           storageClass.Parameters,
		PersistentVolumeList: *persistentVolumeList,
	}
}
