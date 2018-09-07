package statefulset

import (
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type StatefulSetList struct {
	*dataselect.ListMeta

	StatefulSets []StatefulSet
	Pods         []v1.Pod
	Events       []v1.Event
}

// StatefulSet is a presentation layer view of Kubernetes Stateful Set resource. This means it is
// Stateful Set plus additional augmented data we can get from other sources (like services that
// target the same pods).
type StatefulSet struct {
	api.ObjectMeta
	api.TypeMeta

	// Aggregate information about pods belonging to this Pet Set.
	Pods common.PodInfo `json:"pods"`

	// Container images of the Stateful Set.
	ContainerImages []string `json:"containerImages"`

	// Init container images of the Stateful Set.
	InitContainerImages []string `json:"initContainerImages"`
}

func (man *SStatefuleSetManager) List(req *common.Request) (common.ListResource, error) {
	return GetStatefulSetList(req.GetK8sClient(), req.GetNamespaceQuery(), req.ToQuery())
}

// GetStatefulSetList returns a list of all Stateful Sets in the cluster.
func GetStatefulSetList(client kubernetes.Interface, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*StatefulSetList, error) {
	log.Infof("Getting list of all pet sets in the cluster")

	channels := &common.ResourceChannels{
		StatefulSetList: common.GetStatefulSetListChannel(client, nsQuery),
		PodList:         common.GetPodListChannel(client, nsQuery),
		EventList:       common.GetEventListChannel(client, nsQuery),
	}

	return GetStatefulSetListFromChannels(channels, dsQuery)
}

// GetStatefulSetListFromChannels returns a list of all Stateful Sets in the cluster reading
// required resource list once from the channels.
func GetStatefulSetListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*StatefulSetList, error) {
	statefulSets := <-channels.StatefulSetList.List
	err := <-channels.StatefulSetList.Error
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

	return ToStatefulSetList(statefulSets.Items, pods.Items, events.Items, dsQuery)
}

func ToStatefulSetList(statefulSets []apps.StatefulSet, pods []v1.Pod, events []v1.Event, dsQuery *dataselect.DataSelectQuery) (*StatefulSetList, error) {
	statefulSetList := &StatefulSetList{
		StatefulSets: make([]StatefulSet, 0),
		ListMeta:     dataselect.NewListMeta(),
		Pods:         pods,
		Events:       events,
	}

	err := dataselect.ToResourceList(
		statefulSetList,
		statefulSets,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return statefulSetList, err
}

func GetPodInfo(statefulSet apps.StatefulSet, pods []v1.Pod, events []v1.Event) common.PodInfo {
	matchingPods := common.FilterPodsByControllerRef(&statefulSet, pods)
	podInfo := common.GetPodInfo(statefulSet.Status.Replicas, statefulSet.Spec.Replicas, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)
	return podInfo
}

func (l *StatefulSetList) Append(obj interface{}) {
	statefulSet := obj.(apps.StatefulSet)
	podInfo := GetPodInfo(statefulSet, l.Pods, l.Events)
	l.StatefulSets = append(l.StatefulSets, ToStatefulSet(&statefulSet, &podInfo))
}

func (l *StatefulSetList) GetResponseData() interface{} {
	return l.StatefulSets
}

func ToStatefulSet(statefulSet *apps.StatefulSet, podInfo *common.PodInfo) StatefulSet {
	return StatefulSet{
		ObjectMeta:          api.NewObjectMeta(statefulSet.ObjectMeta),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindStatefulSet),
		ContainerImages:     common.GetContainerImages(&statefulSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&statefulSet.Spec.Template.Spec),
		Pods:                *podInfo,
	}
}