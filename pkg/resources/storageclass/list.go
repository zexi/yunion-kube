package storageclass

import (
	storage "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// StorageClassList holds a list of storage class objects in the cluster.
type StorageClassList struct {
	*common.BaseList
	StorageClasses []StorageClass
}

func (man *SStorageClassManager) List(req *common.Request) (common.ListResource, error) {
	return GetStorageClassList(req.GetK8sClient(), req.GetCluster(), req.ToQuery())
}

// GetStorageClassList returns a list of all storage class objects in the cluster.
func GetStorageClassList(client kubernetes.Interface, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery) (*StorageClassList, error) {
	log.Infof("Getting list of storage classes in the cluster")

	channels := &common.ResourceChannels{
		StorageClassList: common.GetStorageClassListChannel(client),
	}

	return GetStorageClassListFromChannels(channels, dsQuery, cluster)
}

// GetStorageClassListFromChannels returns a list of all storage class objects in the cluster.
func GetStorageClassListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*StorageClassList, error) {
	storageClasses := <-channels.StorageClassList.List
	err := <-channels.StorageClassList.Error
	if err != nil {
		return nil, err
	}

	return toStorageClassList(storageClasses.Items, dsQuery, cluster)
}

func toStorageClassList(storageClasses []storage.StorageClass, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*StorageClassList, error) {
	storageClassList := &StorageClassList{
		BaseList:       common.NewBaseList(cluster),
		StorageClasses: make([]StorageClass, 0),
	}

	err := dataselect.ToResourceList(
		storageClassList,
		storageClasses,
		dataselect.NewResourceDataCell,
		dsQuery)

	return storageClassList, err
}

func (l *StorageClassList) Append(obj interface{}) {
	class := obj.(storage.StorageClass)
	l.StorageClasses = append(l.StorageClasses, ToStorageClass(&class, l.GetCluster()))
}

func (l *StorageClassList) GetResponseData() interface{} {
	return l.StorageClasses
}

func ToStorageClass(storageClass *storage.StorageClass, cluster api.ICluster) StorageClass {
	return StorageClass{
		ObjectMeta:  api.NewObjectMetaV2(storageClass.ObjectMeta, cluster),
		TypeMeta:    api.NewTypeMeta(api.ResourceKindStorageClass),
		Provisioner: storageClass.Provisioner,
		Parameters:  storageClass.Parameters,
	}
}
