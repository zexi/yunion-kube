package statefulset

import (
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	ds "yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// StatefulSetDetail is a presentation layer view of Kubernetes Stateful Set resource. This means it is Stateful
// Set plus additional augmented data we can get from other sources (like services that target the same pods).
type StatefulSetDetail struct {
	api.ObjectMeta
	api.TypeMeta
	PodInfo             common.PodInfo `json:"podInfo"`
	PodList             []pod.Pod      `json:"pods"`
	ContainerImages     []string       `json:"containerImages"`
	InitContainerImages []string       `json:"initContainerImages"`
	EventList           []common.Event `json:"events"`
	Status              string         `json:"status"`
}

func (man *SStatefuleSetManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetStatefulSetDetail(req.GetK8sClient(), namespace, id)
}

// GetStatefulSetDetail gets Stateful Set details.
func GetStatefulSetDetail(client kubernetes.Interface, namespace, name string) (*StatefulSetDetail, error) {
	log.Printf("Getting details of %s statefulset in %s namespace", name, namespace)

	ss, err := client.AppsV1beta2().StatefulSets(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	podList, err := GetStatefulSetPods(client, ds.DefaultDataSelect, name, namespace)
	if err != nil {
		return nil, err
	}

	podInfo, err := getStatefulSetPodInfo(client, ss)
	if err != nil {
		return nil, err
	}

	events, err := event.GetResourceEvents(client, ds.DefaultDataSelect, ss.Namespace, ss.Name)
	if err != nil {
		return nil, err
	}

	ssDetail := getStatefulSetDetail(ss, *events, *podList, *podInfo)
	return &ssDetail, nil
}

func getStatefulSetDetail(statefulSet *apps.StatefulSet, eventList common.EventList, podList pod.PodList, podInfo common.PodInfo) StatefulSetDetail {
	return StatefulSetDetail{
		ObjectMeta:          api.NewObjectMeta(statefulSet.ObjectMeta),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindStatefulSet),
		ContainerImages:     common.GetContainerImages(&statefulSet.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&statefulSet.Spec.Template.Spec),
		PodInfo:             podInfo,
		PodList:             podList.Pods,
		EventList:           eventList.Events,
		Status:              podInfo.GetStatus(),
	}
}

// GetStatefulSetPods return list of pods targeting pet set.
func GetStatefulSetPods(client kubernetes.Interface, dsQuery *ds.DataSelectQuery, name, namespace string) (*pod.PodList, error) {

	log.Infof("Getting replication controller %s pods in namespace %s", name, namespace)

	pods, err := getRawStatefulSetPods(client, name, namespace)
	if err != nil {
		return nil, err
	}

	events, err := event.GetPodsEvents(client, namespace, pods)
	if err != nil {
		return nil, err
	}

	return pod.ToPodList(pods, events, dsQuery)
}

// getRawStatefulSetPods return array of api pods targeting pet set with given name.
func getRawStatefulSetPods(client kubernetes.Interface, name, namespace string) ([]v1.Pod, error) {
	statefulSet, err := client.AppsV1beta1().StatefulSets(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannel(client, common.NewSameNamespaceQuery(namespace)),
	}

	podList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	return common.FilterPodsByControllerRef(statefulSet, podList.Items), nil
}

// Returns simple info about pods(running, desired, failing, etc.) related to given pet set.
func getStatefulSetPodInfo(client kubernetes.Interface, statefulSet *apps.StatefulSet) (*common.PodInfo, error) {
	pods, err := getRawStatefulSetPods(client, statefulSet.Name, statefulSet.Namespace)
	if err != nil {
		return nil, err
	}

	podInfo := common.GetPodInfo(statefulSet.Status.Replicas, statefulSet.Spec.Replicas, pods)
	return &podInfo, nil
}
