package persistentvolumeclaim

import (
	"strings"

	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// PersistentVolumeClaimDetail provides the presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaimDetail struct {
	api.ObjectMeta
	api.TypeMeta
	Status       v1.PersistentVolumeClaimPhase   `json:"status"`
	Volume       string                          `json:"volume"`
	Capacity     v1.ResourceList                 `json:"capacity"`
	AccessModes  []v1.PersistentVolumeAccessMode `json:"accessModes"`
	StorageClass *string                         `json:"storageClass"`
}

func (man *SPersistentVolumeClaimManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetPersistentVolumeClaimDetail(req.GetK8sClient(), req.GetNamespaceQuery().ToRequestParam(), id)
}

// GetPersistentVolumeClaimDetail returns detailed information about a persistent volume claim
func GetPersistentVolumeClaimDetail(client kubernetes.Interface, namespace string, name string) (*PersistentVolumeClaimDetail, error) {
	log.Infof("Getting details of %s persistent volume claim", name)

	rawPersistentVolumeClaim, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(name, metaV1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return getPersistentVolumeClaimDetail(rawPersistentVolumeClaim), nil
}

func getPersistentVolumeClaimDetail(persistentVolumeClaim *v1.PersistentVolumeClaim) *PersistentVolumeClaimDetail {

	return &PersistentVolumeClaimDetail{
		ObjectMeta:   api.NewObjectMeta(persistentVolumeClaim.ObjectMeta),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindPersistentVolumeClaim),
		Status:       persistentVolumeClaim.Status.Phase,
		Volume:       persistentVolumeClaim.Spec.VolumeName,
		Capacity:     persistentVolumeClaim.Status.Capacity,
		AccessModes:  persistentVolumeClaim.Spec.AccessModes,
		StorageClass: persistentVolumeClaim.Spec.StorageClassName,
	}
}

// GetPodPersistentVolumeClaims gets persistentvolumeclaims that are associated with this pod.
func GetPodPersistentVolumeClaims(client kubernetes.Interface, namespace string, podName string, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(podName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	claimNames := make([]string, 0)
	if pod.Spec.Volumes != nil && len(pod.Spec.Volumes) > 0 {
		for _, v := range pod.Spec.Volumes {
			persistentVolumeClaim := v.PersistentVolumeClaim
			if persistentVolumeClaim != nil {
				claimNames = append(claimNames, persistentVolumeClaim.ClaimName)
			}
		}
	}

	if len(claimNames) > 0 {
		channels := &common.ResourceChannels{
			PersistentVolumeClaimList: common.GetPersistentVolumeClaimListChannel(
				client, common.NewSameNamespaceQuery(namespace)),
		}

		persistentVolumeClaimList := <-channels.PersistentVolumeClaimList.List

		err = <-channels.PersistentVolumeClaimList.Error
		if err != nil {
			return nil, err
		}

		podPersistentVolumeClaims := make([]v1.PersistentVolumeClaim, 0)
		for _, pvc := range persistentVolumeClaimList.Items {
			for _, claimName := range claimNames {
				if strings.Compare(claimName, pvc.Name) == 0 {
					podPersistentVolumeClaims = append(podPersistentVolumeClaims, pvc)
					break
				}
			}
		}

		log.Infof("Found %d persistentvolumeclaims related to %s pod",
			len(podPersistentVolumeClaims), podName)

		return toPersistentVolumeClaimList(podPersistentVolumeClaims, dsQuery)
	}

	log.Infof("No persistentvolumeclaims found related to %s pod", podName)

	// No ClaimNames found in Pod details, return empty response.
	return &PersistentVolumeClaimList{}, nil
}
