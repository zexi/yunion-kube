package replicaset

import (
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	client "k8s.io/client-go/kubernetes"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// ReplicaSet is a presentation layer view of Kubernetes Replica Set resource. This means
// it is Replica Set plus additional augmented data we can get from other sources
// (like services that target the same pods).
type ReplicaSet struct {
	api.ObjectMeta
	api.TypeMeta

	// Aggregate information about pods belonging to this Replica Set.
	Pods common.PodInfo `json:"pods"`

	// Container images of the Replica Set.
	ContainerImages []string `json:"containerImages"`

	// Init Container images of the Replica Set.
	InitContainerImages []string `json:"initContainerImages"`
}

// ToReplicaSet converts replica set api object to replica set model object.
func ToReplicaSet(replicaSet *apps.ReplicaSet, podInfo *common.PodInfo, cluster api.ICluster) ReplicaSet {
	return ReplicaSet{
		ObjectMeta:          api.NewObjectMeta(replicaSet.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindReplicaSet),
		ContainerImages:     common.GetContainerImages(&replicaSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&replicaSet.Spec.Template.Spec),
		Pods:                *podInfo,
	}
}

type ReplicaSetList struct {
	*common.BaseList
	ReplicaSets []ReplicaSet
	events      []v1.Event
	pods        []v1.Pod
	// Basic information about resources status on the list.
	Status common.ResourceStatus
}

// GetReplicaSetList returns a list of all Replica Sets in the cluster.
func GetReplicaSetList(client client.Interface, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ReplicaSetList, error) {
	log.Infof("Getting list of all replica sets in the cluster")

	channels := &common.ResourceChannels{
		ReplicaSetList: common.GetReplicaSetListChannel(client, nsQuery),
		PodList:        common.GetPodListChannel(client, nsQuery),
		EventList:      common.GetEventListChannel(client, nsQuery),
	}

	return GetReplicaSetListFromChannels(channels, dsQuery, cluster)
}

// GetReplicaSetListFromChannels returns a list of all Replica Sets in the cluster
// reading required resource list once from the channels.
func GetReplicaSetListFromChannels(channels *common.ResourceChannels,
	dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ReplicaSetList, error) {

	replicaSets := <-channels.ReplicaSetList.List
	err := <-channels.ReplicaSetList.Error
	if err != nil {
		return nil, err
	}

	pods := <-channels.PodList.List
	err = <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	events := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	rsList, err := ToReplicaSetList(replicaSets.Items, pods.Items, events.Items, dsQuery, cluster)
	if err != nil {
		return nil, err
	}
	rsList.Status = getStatus(replicaSets, pods.Items, events.Items)
	return rsList, nil
}

func (l *ReplicaSetList) Append(obj interface{}) {
	replicaSet := obj.(apps.ReplicaSet)
	pods := l.pods
	events := l.events
	matchingPods := common.FilterPodsByControllerRef(&replicaSet, pods)
	podInfo := common.GetPodInfo(replicaSet.Status.Replicas, replicaSet.Spec.Replicas,
		matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)

	l.ReplicaSets = append(l.ReplicaSets, ToReplicaSet(&replicaSet, &podInfo, l.GetCluster()))
}

// ToReplicaSetList creates paginated list of Replica Set model
// objects based on Kubernetes Replica Set objects array and related resources arrays.
func ToReplicaSetList(replicaSets []apps.ReplicaSet, pods []v1.Pod, events []v1.Event, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ReplicaSetList, error) {

	replicaSetList := &ReplicaSetList{
		BaseList:    common.NewBaseList(cluster),
		ReplicaSets: make([]ReplicaSet, 0),
		events:      events,
		pods:        pods,
	}

	err := dataselect.ToResourceList(
		replicaSetList,
		replicaSets,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return replicaSetList, err
}

func getStatus(list *apps.ReplicaSetList, pods []v1.Pod, events []v1.Event) common.ResourceStatus {
	info := common.ResourceStatus{}
	if list == nil {
		return info
	}

	for _, rs := range list.Items {
		matchingPods := common.FilterPodsByControllerRef(&rs, pods)
		podInfo := common.GetPodInfo(rs.Status.Replicas, rs.Spec.Replicas, matchingPods)
		warnings := event.GetPodsEventWarnings(events, matchingPods)

		if len(warnings) > 0 {
			info.Failed++
		} else if podInfo.Pending > 0 {
			info.Pending++
		} else {
			info.Running++
		}
	}

	return info
}
