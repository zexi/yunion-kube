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
	*common.BaseList

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
	Pods common.PodInfo `json:"podsInfo"`

	// Container images of the Stateful Set.
	ContainerImages []string `json:"containerImages"`

	// Init container images of the Stateful Set.
	InitContainerImages []string          `json:"initContainerImages"`
	Status              string            `json:"status"`
	Selector            map[string]string `json:"selector"`
}

func (man *SStatefuleSetManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetK8sClient(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SStatefuleSetManager) ListV2(client kubernetes.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetStatefulSetList(client, cluster, nsQuery, dsQuery)
}

// GetStatefulSetList returns a list of all Stateful Sets in the cluster.
func GetStatefulSetList(client kubernetes.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*StatefulSetList, error) {
	log.Infof("Getting list of all pet sets in the cluster")

	channels := &common.ResourceChannels{
		StatefulSetList: common.GetStatefulSetListChannel(client, nsQuery),
		PodList:         common.GetPodListChannel(client, nsQuery),
		EventList:       common.GetEventListChannel(client, nsQuery),
	}

	return GetStatefulSetListFromChannels(cluster, channels, dsQuery)
}

// GetStatefulSetListFromChannels returns a list of all Stateful Sets in the cluster reading
// required resource list once from the channels.
func GetStatefulSetListFromChannels(cluster api.ICluster, channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*StatefulSetList, error) {
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

	return ToStatefulSetList(statefulSets.Items, pods.Items, events.Items, dsQuery, cluster)
}

func ToStatefulSetList(statefulSets []apps.StatefulSet, pods []v1.Pod, events []v1.Event, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*StatefulSetList, error) {
	statefulSetList := &StatefulSetList{
		BaseList:     common.NewBaseList(cluster),
		StatefulSets: make([]StatefulSet, 0),
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
	l.StatefulSets = append(l.StatefulSets, ToStatefulSet(&statefulSet, &podInfo, l.GetCluster()))
}

func (l *StatefulSetList) GetResponseData() interface{} {
	return l.StatefulSets
}

func ToStatefulSet(statefulSet *apps.StatefulSet, podInfo *common.PodInfo, cluster api.ICluster) StatefulSet {
	return StatefulSet{
		ObjectMeta:          api.NewObjectMetaV2(statefulSet.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindStatefulSet),
		ContainerImages:     common.GetContainerImages(&statefulSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&statefulSet.Spec.Template.Spec),
		Pods:                *podInfo,
		Status:              podInfo.GetStatus(),
		Selector:            statefulSet.Spec.Selector.MatchLabels,
	}
}
