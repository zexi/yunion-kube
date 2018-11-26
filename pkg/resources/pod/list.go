package pod

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type PodStatus struct {
	Status          string              `json:"status"`
	PodPhase        v1.PodPhase         `json:"podPhase"`
	ContainerStates []v1.ContainerState `json:"containerStates"`
}

// Pod is a presentation layer view of Pod resource. This means it is Pod plus additional augmented data
// we can get from other sources (like services that target it).
type Pod struct {
	api.ObjectMeta
	api.TypeMeta

	// More info on pod status
	PodStatus

	PodIP string `json:"podIP"`
	// Count of containers restarts
	RestartCount int32 `json:"restartCount"`

	// Pod warning events
	Warnings []common.Event `json:"warnings"`

	// Name of the Node this pod runs on
	NodeName string `json:"nodeName"`
}

type PodList struct {
	*dataselect.ListMeta
	Pods   []Pod
	Events []v1.Event
}

func (l PodList) GetPods() []Pod {
	return l.Pods
}

func (l *PodList) GetResponseData() interface{} {
	return l.Pods
}

func (man *SPodManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetK8sClient(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SPodManager) ListV2(k8sCli kubernetes.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetPodList(k8sCli, nsQuery, dsQuery)
}

func (man *SPodManager) GetPodList(k8sCli kubernetes.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*PodList, error) {
	log.Infof("Getting list of all pods in the cluster")
	channels := &common.ResourceChannels{
		PodList:   common.GetPodListChannelWithOptions(k8sCli, nsQuery, metaV1.ListOptions{}),
		EventList: common.GetEventListChannel(k8sCli, nsQuery),
	}
	return GetPodListFromChannels(channels, dsQuery)
}

func (l *PodList) Append(obj interface{}) {
	pod := obj.(v1.Pod)
	warnings := event.GetPodsEventWarnings(l.Events, []v1.Pod{pod})
	l.Pods = append(l.Pods, ToPod(pod, warnings))
}

func GetPodListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*PodList, error) {
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

	podList, err := ToPodList(pods.Items, eventList.Items, dsQuery)
	return podList, err
}

func ToPodList(pods []v1.Pod, events []v1.Event, dsQuery *dataselect.DataSelectQuery) (*PodList, error) {
	podList := &PodList{
		ListMeta: dataselect.NewListMeta(),
		Pods:     make([]Pod, 0),
		Events:   events,
	}
	err := dataselect.ToResourceList(
		podList,
		pods,
		dataselect.NewNamespacePodStatusDataCell,
		dsQuery)
	return podList, err
}
