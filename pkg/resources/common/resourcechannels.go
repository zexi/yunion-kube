package common

import (
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
}

// PodListChannel is a list and error channels to Nodes
type PodListChannel struct {
	List  chan *v1.PodList
	Error chan error
}

func GetPodListChannel(client client.Interface, nsQuery *NamespaceQuery, numReads int) PodListChannel {
	return GetPodListChannelWithOptions(client, nsQuery, api.ListEverything, numReads)
}

func GetPodListChannelWithOptions(client client.Interface, nsQuery *NamespaceQuery, options metaV1.ListOptions, numReads int) PodListChannel {
	channel := PodListChannel{
		List:  make(chan *v1.PodList, numReads),
		Error: make(chan error, numReads),
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
		for i := 0; i < numReads; i++ {
			channel.List <- list
			channel.Error <- err
		}
	}()

	return channel
}
