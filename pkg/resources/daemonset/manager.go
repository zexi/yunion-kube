package daemonset

import (
	"k8s.io/api/core/v1"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"

	apps "k8s.io/api/apps/v1"
	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

var (
	DaemonSetManager *SDaemonSetManager
)

type SDaemonSetManager struct {
	*resources.SNamespaceResourceManager
}

/*func (m *SDaemonSetManager) IsRawResource() bool {
	return false
}*/

func init() {
	DaemonSetManager = &SDaemonSetManager{
		resources.NewNamespaceResourceManager("daemonset", "daemonsets"),
	}
	resources.KindManagerMap.Register(apis.KindNameDaemonSet, DaemonSetManager)
}

func (m *SDaemonSetManager) List(req *common.Request) (common.ListResource, error) {
	return m.GetDaemonSetList(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (m *SDaemonSetManager) Get(req *common.Request, id string) (interface{}, error) {
	return m.GetDaemonSetDetail(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery().ToRequestParam(), id)
}

func (m *SDaemonSetManager) GetDetails(cli *client.CacheFactory, cluster apis.ICluster, namespace, name string) (interface{}, error) {
	return m.GetDaemonSetDetail(cli, cluster, namespace, name)
}

func (m *SDaemonSetManager) GetDaemonSetDetail(cli *client.CacheFactory, cluster apis.ICluster, namespace, name string) (*apis.DaemonSet, error) {
	ds, err := cli.DaemonSetLister().DaemonSets(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		EventList: common.GetEventListChannel(cli, common.NewSameNamespaceQuery(namespace)),
		PodList:   common.GetPodListChannel(cli, common.NewSameNamespaceQuery(namespace)),
	}

	events := <-channels.EventList.List
	if err := <-channels.EventList.Error; err != nil {
		return nil, err
	}
	pods := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}
	return ToDaemonSet(ds, pods, events, cluster), nil
}

type DaemonSetList struct {
	*common.BaseList
	ds     []*apis.DaemonSet
	events []*v1.Event
	pods   []*v1.Pod
}

func (m *SDaemonSetManager) GetDaemonSetList(
	indexer *client.CacheFactory, cluster apis.ICluster,
	nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery,
) (*DaemonSetList, error) {
	channels := &common.ResourceChannels{
		DaemonSetList: common.GetDaemonSetListChannel(indexer, nsQuery),
		ServiceList:   common.GetServiceListChannel(indexer, nsQuery),
		PodList:       common.GetPodListChannel(indexer, nsQuery),
		EventList:     common.GetEventListChannel(indexer, nsQuery),
	}
	return GetDaemonSetListFromChannels(channels, dsQuery, cluster)
}

func GetDaemonSetListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster apis.ICluster) (*DaemonSetList, error) {

	ds := <-channels.DaemonSetList.List
	err := <-channels.DaemonSetList.Error
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
	dsList := &DaemonSetList{
		BaseList: common.NewBaseList(cluster),
		ds:       make([]*apis.DaemonSet, 0),
		pods:     pods,
		events:   events,
	}
	err = dataselect.ToResourceList(
		dsList,
		ds,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)
	return dsList, err
}

func (l *DaemonSetList) Append(obj interface{}) {
	l.ds = append(l.ds, ToDaemonSet(
		obj.(*apps.DaemonSet),
		l.pods,
		l.events,
		l.GetCluster(),
	))
}

func (l *DaemonSetList) GetResponseData() interface{} {
	return l.ds
}

func ToDaemonSet(ds *apps.DaemonSet, pods []*v1.Pod, events []*v1.Event, cluster apis.ICluster) *apis.DaemonSet {
	matchingPods := common.FilterPodsByControllerRef(ds, pods)
	podInfo := common.GetPodInfo(ds.Status.CurrentNumberScheduled, &ds.Status.DesiredNumberScheduled, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)

	return &apis.DaemonSet{
		ObjectMeta:          apis.NewObjectMeta(ds.ObjectMeta, cluster),
		TypeMeta:            apis.NewTypeMeta(ds.TypeMeta),
		ContainerImages:     common.GetContainerImages(&ds.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&ds.Spec.Template.Spec),
		PodInfo:             podInfo,
		DaemonSetStatus:     *getters.GetDaemonsetStatus(&podInfo, *ds),
		Selector:            ds.Spec.Selector,
	}
}
