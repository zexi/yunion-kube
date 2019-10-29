package pod

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

type PodList struct {
	*common.BaseList
	Pods       []api.Pod
	Events     []*v1.Event
	ConfigMaps []*v1.ConfigMap
	Secrets    []*v1.Secret
}

func (l PodList) GetPods() []api.Pod {
	return l.Pods
}

func (l *PodList) GetResponseData() interface{} {
	return l.Pods
}

func (man *SPodManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SPodManager) ListV2(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery,
) (common.ListResource, error) {
	return man.GetPodList(indexer, nsQuery, dsQuery, cluster)
}

func (man *SPodManager) GetPodList(
	indexer *client.CacheFactory,
	nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery,
	cluster api.ICluster,
) (*PodList, error) {
	log.Infof("Getting list of all pods in the cluster")
	channels := &common.ResourceChannels{
		PodList:       common.GetPodListChannelWithOptions(indexer, nsQuery, labels.Everything()),
		ConfigMapList: common.GetConfigMapListChannel(indexer, nsQuery),
		SecretList:    common.GetSecretListChannel(indexer, nsQuery),
		EventList:     common.GetEventListChannel(indexer, nsQuery),
	}
	return GetPodListFromChannels(indexer, channels, dsQuery, cluster)
}

func (l *PodList) Append(obj interface{}) {
	pod := obj.(*v1.Pod)
	warnings := event.GetPodsEventWarnings(l.Events, []*v1.Pod{pod})
	l.Pods = append(l.Pods, ToPod(pod, warnings, l.ConfigMaps, l.Secrets, l.GetCluster()))
}

func GetPodListFromChannels(indexer *client.CacheFactory, channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*PodList, error) {
	pods := <-channels.PodList.List
	err := <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	eventList := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	cfgs := <-channels.ConfigMapList.List
	err = <-channels.ConfigMapList.Error
	if err != nil {
		return nil, err
	}

	secrets := <-channels.SecretList.List
	if err != nil {
		return nil, err
	}

	podList, err := ToPodList(pods, eventList, secrets, cfgs, dsQuery, cluster)
	return podList, err
}

func ToPodListByIndexer(
	indexer *client.CacheFactory,
	namespace string,
	dsQuery *dataselect.DataSelectQuery,
	selector labels.Selector,
	cluster api.ICluster,
) (*PodList, error) {
	pods, err := indexer.PodLister().Pods(namespace).List(selector)
	if err != nil {
		return nil, err
	}
	return ToPodListByIndexerV2(indexer, pods, namespace, dsQuery, selector, cluster)
}

func ToPodListByIndexerV2(
	indexer *client.CacheFactory,
	pods []*v1.Pod,
	namespace string,
	dsQuery *dataselect.DataSelectQuery,
	selector labels.Selector,
	cluster api.ICluster,
) (*PodList, error) {
	events, err := event.GetPodsEvents(indexer, namespace, pods)
	if err != nil {
		return nil, err
	}
	secrets, err := indexer.SecretLister().Secrets(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	cfgs, err := indexer.ConfigMapLister().ConfigMaps(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	return ToPodList(pods, events, secrets, cfgs, dsQuery, cluster)
}

func ToPodList(
	pods []*v1.Pod,
	events []*v1.Event,
	secrets []*v1.Secret,
	cfgs []*v1.ConfigMap,
	dsQuery *dataselect.DataSelectQuery,
	cluster api.ICluster) (*PodList, error) {
	podList := &PodList{
		BaseList:   common.NewBaseList(cluster),
		Pods:       make([]api.Pod, 0),
		Events:     events,
		ConfigMaps: cfgs,
		Secrets:    secrets,
	}
	err := dataselect.ToResourceList(
		podList,
		pods,
		dataselect.NewNamespacePodStatusDataCell,
		dsQuery)
	return podList, err
}
