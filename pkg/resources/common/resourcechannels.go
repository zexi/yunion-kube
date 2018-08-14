package common

import (
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// ResourceChannels struct holds channels to resource lists. Each list channel is paired with
// an error channel which *must* be read sequentially: first read the list channel and then the error channel.
type ResourceChannels struct {
	// List and error channels to Pods
	PodList PodListChannel

	// List and error channels to Services
	ServiceList ServiceListChannel
}

// PodListChannel is a list and error channels to Nodes
type PodListChannel struct {
	List  chan *v1.PodList
	Error chan error
}

func GetPodListChannel(client client.Interface, nsQuery *NamespaceQuery) PodListChannel {
	return GetPodListChannelWithOptions(client, nsQuery, api.ListEverything)
}

func GetPodListChannelWithOptions(client client.Interface, nsQuery *NamespaceQuery, options metaV1.ListOptions) PodListChannel {
	channel := PodListChannel{
		List:  make(chan *v1.PodList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Pods(nsQuery.ToRequestParam()).List(options)
		var filteredItems []v1.Pod
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type ServiceListChannel struct {
	List  chan *v1.ServiceList
	Error chan error
}

func GetServiceListChannel(client client.Interface, nsQuery *NamespaceQuery) ServiceListChannel {
	channel := ServiceListChannel{
		List:  make(chan *v1.ServiceList),
		Error: make(chan error),
	}
	go func() {
		list, err := client.CoreV1().Services(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []v1.Service
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()
	return channel
}

type DeploymentListChannel struct {
	List  chan *apps.DeploymentList
	Error chan error
}

func GetDeploymentListChannel(client client.Interface,
	nsQuery *NamespaceQuery) DeploymentListChannel {
	channel := DeploymentListChannel{
		List:  make(chan *apps.DeploymentList),
		Error: make(chan error),
	}
	go func() {
		list, err := client.AppsV1beta2().Deployments(nsQuery.ToRequestParam()).
			List(api.ListEverything)
		var filteredItems []apps.Deployment
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}
