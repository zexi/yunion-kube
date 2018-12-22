package persistentvolumeclaim

import (
	"strings"

	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

// PersistentVolumeClaimDetail provides the presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaimDetail struct {
	PersistentVolumeClaim
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

	pods, err := client.CoreV1().Pods(namespace).List(metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return getPersistentVolumeClaimDetail(rawPersistentVolumeClaim, pods.Items), nil
}

func getPersistentVolumeClaimDetail(pvc *v1.PersistentVolumeClaim, pods []v1.Pod) *PersistentVolumeClaimDetail {
	return &PersistentVolumeClaimDetail{
		PersistentVolumeClaim: toPersistentVolumeClaim(*pvc, pods),
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

		pvcs := []PersistentVolumeClaim{}
		for _, pvc := range podPersistentVolumeClaims {
			pvcs = append(pvcs, toPersistentVolumeClaim(pvc, []v1.Pod{*pod}))
		}
		return toPersistentVolumeClaimList(pvcs, dsQuery)
	}

	log.Infof("No persistentvolumeclaims found related to %s pod", podName)

	// No ClaimNames found in Pod details, return empty response.
	return &PersistentVolumeClaimList{}, nil
}
