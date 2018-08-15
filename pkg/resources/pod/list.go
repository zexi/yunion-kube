package pod

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
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
	//Warnings []resources.Event `json:"warnings"`

	// Name of the Node this pod runs on
	NodeName string `json:"nodeName"`
}

// ToListItem dynamic called by common.ToListJsonData
func (p Pod) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(p)
}

type PodList struct {
	*dataselect.ListMeta
	pods []Pod
}

func (l *PodList) GetResponseData() interface{} {
	return l.pods
}

func (man *SPodManager) AllowListItems(req *common.Request) bool {
	return req.AllowListItems()
}

func (man *SPodManager) List(req *common.Request) (common.ListResource, error) {
	return man.GetPodList(req.GetK8sClient(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SPodManager) GetPodList(k8sCli kubernetes.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*PodList, error) {
	log.Infof("Getting list of all pods in the cluster")
	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(k8sCli, nsQuery, metaV1.ListOptions{}),
	}
	return GetPodListFromChannels(channels, dsQuery)
}

func (l *PodList) Append(obj interface{}) {
	l.pods = append(l.pods, ToPod(obj.(v1.Pod)))
}

func GetPodListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*PodList, error) {
	pods := <-channels.PodList.List
	err := <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	podList := &PodList{
		ListMeta: dataselect.NewListMeta(),
		pods:     make([]Pod, 0),
	}
	log.Debugf("====Get pods: %#v", pods.Items)
	err = dataselect.ToResourceList(
		podList,
		pods.Items,
		dataselect.NewNamespacePodStatusDataCell,
		dsQuery)
	return podList, err
}
