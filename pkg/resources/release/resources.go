package release

import (
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/proto/hapi/release"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/ingress"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	"yunion.io/x/yunion-kube/pkg/resources/service"
	"yunion.io/x/yunion-kube/pkg/resources/statefulset"
)

type Resources struct {
	Pods         []pod.Pod                 `json:"pods"`
	Services     []service.Service         `json:"services"`
	Ingresses    []ingress.Ingress         `json:"ingresses"`
	StatefulSets []statefulset.StatefulSet `json:"statefulsets"`
}

func GetReleaseResources(cli kubernetes.Interface, rls *release.Release) (*Resources, error) {
	nsQuery := common.NewSameNamespaceQuery(rls.Namespace)
	labelsMap := map[string]string{
		"release": rls.Name,
	}
	listOpt := metav1.ListOptions{
		FieldSelector: fields.Everything().String(),
		LabelSelector: labels.Set(labelsMap).AsSelector().String(),
	}
	channels := &common.ResourceChannels{
		PodList:         common.GetPodListChannelWithOptions(cli, nsQuery, listOpt),
		ServiceList:     common.GetServiceListChannelWithOptions(cli, nsQuery, listOpt),
		IngressList:     common.GetIngressListChannelWithOptions(cli, nsQuery, listOpt),
		StatefulSetList: common.GetStatefulSetListChannelWithOptions(cli, nsQuery, listOpt),
		EventList:       common.GetEventListChannel(cli, nsQuery),
	}
	pods := <-channels.PodList.List
	err := <-channels.PodList.Error
	if err != nil {
		return nil, err
	}
	svcs := <-channels.ServiceList.List
	err = <-channels.ServiceList.Error
	if err != nil {
		return nil, err
	}
	ings := <-channels.IngressList.List
	err = <-channels.IngressList.Error
	if err != nil {
		return nil, err
	}
	states := <-channels.StatefulSetList.List
	err = <-channels.StatefulSetList.Error
	if err != nil {
		return nil, err
	}
	events := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}
	return transforResource(pods, svcs, ings, states, events), nil
}

func transforResource(
	pods *v1.PodList,
	svcs *v1.ServiceList,
	ings *extensions.IngressList,
	statefulSets *apps.StatefulSetList,
	events *v1.EventList,
) *Resources {
	res := &Resources{
		Pods:         make([]pod.Pod, len(pods.Items)),
		Services:     make([]service.Service, len(svcs.Items)),
		Ingresses:    make([]ingress.Ingress, len(ings.Items)),
		StatefulSets: make([]statefulset.StatefulSet, len(statefulSets.Items)),
	}
	for i, item := range pods.Items {
		res.Pods[i] = pod.ToPod(item)
	}
	for i, item := range svcs.Items {
		res.Services[i] = service.ToService(item)
	}
	for i, item := range ings.Items {
		res.Ingresses[i] = ingress.ToIngress(&item)
	}
	for i, item := range statefulSets.Items {
		podInfo := statefulset.GetPodInfo(item, pods.Items, events.Items)
		res.StatefulSets[i] = statefulset.ToStatefulSet(&item, &podInfo)
	}
	return res
}
