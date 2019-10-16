package persistentvolumeclaim

import (
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// PersistentVolumeClaimDetail provides the presentation layer view of Kubernetes Persistent Volume Claim resource.
type PersistentVolumeClaimDetail struct {
	PersistentVolumeClaim
}

func (man *SPersistentVolumeClaimManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetPersistentVolumeClaimDetail(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery().ToRequestParam(), id)
}

// GetPersistentVolumeClaimDetail returns detailed information about a persistent volume claim
func GetPersistentVolumeClaimDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace string, name string) (*PersistentVolumeClaimDetail, error) {
	log.Infof("Getting details of %s persistent volume claim", name)

	rawPersistentVolumeClaim, err := indexer.PVCLister().PersistentVolumeClaims(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	pods, err := indexer.PodLister().Pods(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return getPersistentVolumeClaimDetail(cluster, rawPersistentVolumeClaim, pods), nil
}

func getPersistentVolumeClaimDetail(cluster api.ICluster, pvc *v1.PersistentVolumeClaim, pods []*v1.Pod) *PersistentVolumeClaimDetail {
	return &PersistentVolumeClaimDetail{
		PersistentVolumeClaim: toPersistentVolumeClaim(pvc, pods, cluster),
	}
}

// GetPodPersistentVolumeClaims gets persistentvolumeclaims that are associated with this pod.
func GetPodPersistentVolumeClaims(indexer *client.CacheFactory, cluster api.ICluster, namespace string, podName string, dsQuery *dataselect.DataSelectQuery) (*PersistentVolumeClaimList, error) {
	pod, err := indexer.PodLister().Pods(namespace).Get(podName)
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
				indexer, common.NewSameNamespaceQuery(namespace)),
		}

		persistentVolumeClaimList := <-channels.PersistentVolumeClaimList.List

		err = <-channels.PersistentVolumeClaimList.Error
		if err != nil {
			return nil, err
		}

		podPersistentVolumeClaims := make([]*v1.PersistentVolumeClaim, 0)
		for _, pvc := range persistentVolumeClaimList {
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
			pvcs = append(pvcs, toPersistentVolumeClaim(pvc, []*v1.Pod{pod}, cluster))
		}
		return toPersistentVolumeClaimList(pvcs, dsQuery, cluster)
	}

	log.Infof("No persistentvolumeclaims found related to %s pod", podName)

	// No ClaimNames found in Pod details, return empty response.
	return &PersistentVolumeClaimList{}, nil
}
