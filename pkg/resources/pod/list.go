package pod

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/errors"
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
	ObjectMeta api.ObjectMeta `json:"objectMeta"`
	TypeMeta   api.TypeMeta   `json:"typeMeta"`

	// More info on pod status
	PodStatus PodStatus `json:"podStatus"`

	PodIP string `json:"podIP"`
	// Count of containers restarts
	RestartCount int32 `json:"restartCount"`

	// Pod warning events
	//Warnings []resources.Event `json:"warnings"`

	// Name of the Node this pod runs on
	NodeName string `json:"nodeName"`
}

type listItem struct {
	api.ObjectMeta
	PodStatus
	PodIP        string `json:"podIP"`
	NodeName     string `json:"nodeName"`
	RestartCount int32  `json:"restartCount"`
}

func (p Pod) ToListItem() jsonutils.JSONObject {
	item := listItem{
		ObjectMeta:   p.ObjectMeta,
		PodStatus:    p.PodStatus,
		PodIP:        p.PodIP,
		NodeName:     p.NodeName,
		RestartCount: p.RestartCount,
	}
	return jsonutils.Marshal(item)
}

type PodList struct {
	api.ListMeta
	pods []Pod
}

func (l *PodList) GetData() []jsonutils.JSONObject {
	ret := make([]jsonutils.JSONObject, len(l.pods))
	for i, item := range l.pods {
		ret[i] = item.ToListItem()
	}
	return ret
}

func (man *SPodManager) AllowListItems(req *common.Request) bool {
	return req.AllowListItems()
}

func (man *SPodManager) List(k8sCli kubernetes.Interface, req *common.Request) (common.ListResource, error) {
	return man.GetPodList(k8sCli, req)
}

func (man *SPodManager) GetPodList(k8sCli kubernetes.Interface, req *common.Request) (*PodList, error) {
	log.Infof("Getting list of all pods in the cluster")
	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(k8sCli, req.GetNamespace(), metaV1.ListOptions{}, 1),
	}
	return GetPodListFromChannels(channels, req)
}

func GetPodListFromChannels(channels *common.ResourceChannels, req *common.Request) (*PodList, error) {
	pods := <-channels.PodList.List
	err := <-channels.PodList.Error
	nonCriticalErrors, criticalError := errors.HandleError(err)
	if criticalError != nil {
		return nil, criticalError
	}

	podList := ToPodList(pods.Items, nonCriticalErrors, req)
	return &podList, nil
}

func ToPodList(pods []v1.Pod, nonCriticalErrors []error, req *common.Request) PodList {
	podList := PodList{
		pods: make([]Pod, 0),
	}
	log.Debugf("=======Get pods: %v", pods)
	selector := dataselect.GenericDataSelector(toCells(pods), dataselect.NoDataSelect)
	pods = fromCells(selector.Data())
	podList.ListMeta = selector.ListMeta()

	for _, pod := range pods {
		podDetail := toPod(&pod)
		podList.pods = append(podList.pods, podDetail)
	}
	return podList
}

func toPod(pod *v1.Pod) Pod {
	podDetail := Pod{
		ObjectMeta:   api.NewObjectMeta(pod.ObjectMeta),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindPod),
		PodStatus:    getPodStatus(*pod),
		RestartCount: getRestartCount(*pod),
		NodeName:     pod.Spec.NodeName,
		PodIP:        pod.Status.PodIP,
	}
	return podDetail
}
