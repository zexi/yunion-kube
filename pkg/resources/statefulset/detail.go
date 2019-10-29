package statefulset

import (
	"reflect"

	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"yunion.io/x/yunion-kube/pkg/client"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	ds "yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	"yunion.io/x/yunion-kube/pkg/resources/service"
)

func (man *SStatefuleSetManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetStatefulSetDetail(req.GetIndexer(), req.GetCluster(), namespace, id)
}

// GetStatefulSetDetail gets Stateful Set details.
func GetStatefulSetDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*api.StatefulSetDetail, error) {
	log.Printf("Getting details of %s statefulset in %s namespace", name, namespace)

	ss, err := indexer.StatefulSetLister().StatefulSets(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		ServiceList: common.GetServiceListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}
	svcList, err := service.GetServiceListFromChannels(channels, ds.DefaultDataSelect(), cluster)
	if err != nil {
		return nil, err
	}

	podList, err := GetStatefulSetPods(indexer, cluster, ds.DefaultDataSelect(), name, namespace)
	if err != nil {
		return nil, err
	}

	podInfo, err := getStatefulSetPodInfo(indexer, ss)
	if err != nil {
		return nil, err
	}

	events, err := event.GetResourceEvents(indexer, cluster, ds.DefaultDataSelect(), ss.Namespace, ss.Name)
	if err != nil {
		return nil, err
	}

	commonSs := ToStatefulSet(ss, podInfo, cluster)

	// filter services by selector
	podLabel := ss.Spec.Selector.MatchLabels
	svcs := make([]api.Service, 0)
	for _, svc := range svcList.Services {
		if reflect.DeepEqual(svc.Selector, podLabel) {
			svcs = append(svcs, svc)
		}
	}

	ssDetail := getStatefulSetDetail(commonSs, ss, events, podList, podInfo, svcs)
	return &ssDetail, nil
}

func getStatefulSetDetail(
	commonSs api.StatefulSet,
	statefulSet *apps.StatefulSet,
	eventList *common.EventList,
	podList *pod.PodList,
	podInfo *api.PodInfo,
	svcList []api.Service,
) api.StatefulSetDetail {
	return api.StatefulSetDetail{
		StatefulSet: commonSs,
		PodList:     podList.Pods,
		EventList:   eventList.Events,
		ServiceList: svcList,
	}
}

// GetStatefulSetPods return list of pods targeting pet set.
func GetStatefulSetPods(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *ds.DataSelectQuery, name, namespace string) (*pod.PodList, error) {

	log.Infof("Getting replication controller %s pods in namespace %s", name, namespace)

	pods, err := getRawStatefulSetPods(indexer, name, namespace)
	if err != nil {
		return nil, err
	}

	return pod.ToPodListByIndexerV2(indexer, pods, namespace, dsQuery, labels.Everything(), cluster)
}

// getRawStatefulSetPods return array of api pods targeting pet set with given name.
func getRawStatefulSetPods(indexer *client.CacheFactory, name, namespace string) ([]*v1.Pod, error) {
	statefulSet, err := indexer.StatefulSetLister().StatefulSets(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}

	podList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	return common.FilterPodsByControllerRef(statefulSet, podList), nil
}

// Returns simple info about pods(running, desired, failing, etc.) related to given pet set.
func getStatefulSetPodInfo(indexer *client.CacheFactory, statefulSet *apps.StatefulSet) (*api.PodInfo, error) {
	pods, err := getRawStatefulSetPods(indexer, statefulSet.Name, statefulSet.Namespace)
	if err != nil {
		return nil, err
	}

	podInfo := common.GetPodInfo(statefulSet.Status.Replicas, statefulSet.Spec.Replicas, pods)
	return &podInfo, nil
}
