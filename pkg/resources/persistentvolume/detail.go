package persistentvolume

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

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
	return GetPersistentVolumeDetail(req.GetK8sClient(), id)
}

// GetPersistentVolumeDetail returns detailed information about a persistent volume
func GetPersistentVolumeDetail(client kubernetes.Interface, name string) (*PersistentVolumeDetail, error) {
	log.Infof("Getting details of %s persistent volume", name)

	rawPersistentVolume, err := client.CoreV1().PersistentVolumes().Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return getPersistentVolumeDetail(rawPersistentVolume), nil
}

func getPersistentVolumeDetail(persistentVolume *v1.PersistentVolume) *PersistentVolumeDetail {
	return &PersistentVolumeDetail{
		ObjectMeta:             api.NewObjectMeta(persistentVolume.ObjectMeta),
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