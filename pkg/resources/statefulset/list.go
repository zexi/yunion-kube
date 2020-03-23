package statefulset

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

type StatefulSetList struct {
	*common.BaseList

	StatefulSets []api.StatefulSet
	Pods         []*v1.Pod
	Events       []*v1.Event
}

func (man *SStatefuleSetManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SStatefuleSetManager) ListV2(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetStatefulSetList(indexer, cluster, nsQuery, dsQuery)
}

// GetStatefulSetList returns a list of all Stateful Sets in the cluster.
func GetStatefulSetList(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (*StatefulSetList, error) {
	log.Infof("Getting list of all pet sets in the cluster")

	channels := &common.ResourceChannels{
		StatefulSetList: common.GetStatefulSetListChannel(indexer, nsQuery),
		PodList:         common.GetPodListChannel(indexer, nsQuery),
		EventList:       common.GetEventListChannel(indexer, nsQuery),
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

	return ToStatefulSetList(statefulSets, pods, events, dsQuery, cluster)
}

func ToStatefulSetList(statefulSets []*apps.StatefulSet, pods []*v1.Pod, events []*v1.Event, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*StatefulSetList, error) {
	statefulSetList := &StatefulSetList{
		BaseList:     common.NewBaseList(cluster),
		StatefulSets: make([]api.StatefulSet, 0),
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

func GetPodInfo(statefulSet *apps.StatefulSet, pods []*v1.Pod, events []*v1.Event) api.PodInfo {
	matchingPods := common.FilterPodsByControllerRef(statefulSet, pods)
	podInfo := common.GetPodInfo(statefulSet.Status.Replicas, statefulSet.Spec.Replicas, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)
	return podInfo
}

func (l *StatefulSetList) Append(obj interface{}) {
	statefulSet := obj.(*apps.StatefulSet)
	podInfo := GetPodInfo(statefulSet, l.Pods, l.Events)
	l.StatefulSets = append(l.StatefulSets, ToStatefulSet(statefulSet, &podInfo, l.GetCluster()))
}

func (l *StatefulSetList) GetResponseData() interface{} {
	return l.StatefulSets
}

func ToStatefulSet(statefulSet *apps.StatefulSet, podInfo *api.PodInfo, cluster api.ICluster) api.StatefulSet {
	return api.StatefulSet{
		ObjectMeta:          api.NewObjectMeta(statefulSet.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(statefulSet.TypeMeta),
		ContainerImages:     common.GetContainerImages(&statefulSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&statefulSet.Spec.Template.Spec),
		Pods:                *podInfo,
		StatefulSetStatus:   *getters.GetStatefulSetStatus(podInfo, *statefulSet),
		Selector:            statefulSet.Spec.Selector.MatchLabels,
	}
}
