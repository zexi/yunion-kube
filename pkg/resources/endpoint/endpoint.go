package endpoint

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Endpoint struct {
	api.ObjectMeta
	api.TypeMeta

	// Hostname, either as a domain name or IP address.
	Host string `json:"host"`

	// Name of the node the endpoint is located
	NodeName *string `json:"nodeName"`

	// Status of the endpoint
	Ready bool `json:"ready"`

	// Array of endpoint ports
	Ports []v1.EndpointPort `json:"ports"`
}

// GetServiceEndpoints gets list of endpoints targeted by given label selector in given namespace.
func GetServiceEndpoints(client client.Interface, namespace, name string) (*EndpointList, error) {
	serviceEndpoints, err := GetEndpoints(client, namespace, name)
	if err != nil {
		return nil, err
	}

	endpointList := toEndpointList(serviceEndpoints)
	log.Infof("Found %d endpoints related to %s service in %s namespace", len(endpointList.Endpoints), name, namespace)
	return endpointList, nil
}

// GetEndpoints gets endpoints associated to resource with given name.
func GetEndpoints(client client.Interface, namespace, name string) ([]v1.Endpoints, error) {
	fieldSelector, err := fields.ParseSelector("metadata.name" + "=" + name)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		EndpointList: common.GetEndpointListChannelWithOptions(client,
			common.NewSameNamespaceQuery(namespace),
			metaV1.ListOptions{
				LabelSelector: labels.Everything().String(),
				FieldSelector: fieldSelector.String(),
			}),
	}

	endpointList := <-channels.EndpointList.List
	if err := <-channels.EndpointList.Error; err != nil {
		return nil, err
	}

	return endpointList.Items, nil
}

// toEndpoint converts endpoint api Endpoint to Endpoint model object.
func toEndpoint(address v1.EndpointAddress, ports []v1.EndpointPort, ready bool) *Endpoint {
	return &Endpoint{
		TypeMeta: api.NewTypeMeta(api.ResourceKindEndpoint),
		Host:     address.IP,
		Ports:    ports,
		Ready:    ready,
		NodeName: address.NodeName,
	}
}

type EndpointList struct {
	*dataselect.ListMeta
	// List of endpoints
	Endpoints []Endpoint `json:"endpoints"`
}

// toEndpointList converts array of api events to endpoint List structure
func toEndpointList(endpoints []v1.Endpoints) *EndpointList {
	endpointList := EndpointList{
		Endpoints: make([]Endpoint, 0),
		ListMeta:  dataselect.NewListMeta(),
	}

	for _, endpoint := range endpoints {
		for _, subSets := range endpoint.Subsets {
			for _, address := range subSets.Addresses {
				endpointList.Endpoints = append(endpointList.Endpoints, *toEndpoint(address, subSets.Ports, true))
			}
			for _, notReadyAddress := range subSets.NotReadyAddresses {
				endpointList.Endpoints = append(endpointList.Endpoints, *toEndpoint(notReadyAddress, subSets.Ports, false))
			}
		}
	}

	return &endpointList
}