package persistentvolumeclaim

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type PersistentVolumeClaimList struct {
	*dataselect.ListMeta
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
}

func (man *SPersistentVolumeClaimManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetK8sClient(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SPersistentVolumeClaimManager) ListV2(client kubernetes.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetPersistentVolumeClaimList(client, nsQuery, dsQuery)
}

// GetPersistentVolumeClaimList returns a list of all Persistent Volume Claims in the cluster.
func GetPersistentVolumeClaimList(client kubernetes.Interface, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {
	log.Infof("Getting list persistent volumes claims")
	channels := &common.ResourceChannels{
		PersistentVolumeClaimList: common.GetPersistentVolumeClaimListChannel(client, nsQuery),
	}

	return GetPersistentVolumeClaimListFromChannels(channels, nsQuery, dsQuery)
}

// GetPersistentVolumeClaimListFromChannels returns a list of all Persistent Volume Claims in the cluster
// reading required resource list once from the channels.
func GetPersistentVolumeClaimListFromChannels(channels *common.ResourceChannels, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {

	persistentVolumeClaims := <-channels.PersistentVolumeClaimList.List
	err := <-channels.PersistentVolumeClaimList.Error
	if err != nil {
		return nil, err
	}

	return toPersistentVolumeClaimList(persistentVolumeClaims.Items, dsQuery)
}

func toPersistentVolumeClaim(pvc v1.PersistentVolumeClaim) PersistentVolumeClaim {
	return PersistentVolumeClaim{
		ObjectMeta:   api.NewObjectMeta(pvc.ObjectMeta),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindPersistentVolumeClaim),
		Status:       string(pvc.Status.Phase),
		Volume:       pvc.Spec.VolumeName,
		Capacity:     pvc.Status.Capacity,
		AccessModes:  pvc.Spec.AccessModes,
		StorageClass: pvc.Spec.StorageClassName,
	}
}

func toPersistentVolumeClaimList(persistentVolumeClaims []v1.PersistentVolumeClaim, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {
	result := &PersistentVolumeClaimList{
		Items:    make([]PersistentVolumeClaim, 0),
		ListMeta: dataselect.NewListMeta(),
	}

	err := dataselect.ToResourceList(
		result,
		persistentVolumeClaims,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)

	return result, err
}

func (l *PersistentVolumeClaimList) Append(obj interface{}) {
	item := obj.(v1.PersistentVolumeClaim)
	l.Items = append(l.Items, toPersistentVolumeClaim(item))
}

func (l *PersistentVolumeClaimList) GetResponseData() interface{} {
	return l.Items
}
