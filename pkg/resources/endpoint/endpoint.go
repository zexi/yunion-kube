package endpoint

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

// GetServiceEndpoints gets list of endpoints targeted by given label selector in given namespace.
func GetServiceEndpoints(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*EndpointList, error) {
	serviceEndpoints, err := GetEndpoints(indexer, namespace, name)
	if err != nil {
		return nil, err
	}

	endpointList := toEndpointList(serviceEndpoints, cluster)
	log.Infof("Found %d endpoints related to %s service in %s namespace", len(endpointList.Endpoints), name, namespace)
	return endpointList, nil
}

// GetEndpoints gets endpoints associated to resource with given name.
func GetEndpoints(client *client.CacheFactory, namespace, name string) ([]*v1.Endpoints, error) {
	channels := &common.ResourceChannels{
		EndpointList: common.GetEndpointListChannelWithOptions(client,
			common.NewSameNamespaceQuery(namespace),
			labels.Everything(),
		),
	}

	endpointList := <-channels.EndpointList.List
	if err := <-channels.EndpointList.Error; err != nil {
		return nil, err
	}

	rs := make([]*v1.Endpoints, 0)
	for _, ep := range endpointList {
		if ep.Name == name {
			rs = append(rs, ep)
		}
	}

	return rs, nil
}

// toEndpoint converts endpoint api Endpoint to Endpoint model object.
func toEndpoint(ep *v1.Endpoints, address v1.EndpointAddress, ports []v1.EndpointPort, ready bool, cluster api.ICluster) *api.EndpointDetail {
	return &api.EndpointDetail{
		ObjectMeta: api.NewObjectMeta(ep.ObjectMeta, cluster),
		TypeMeta: api.NewTypeMeta(ep.TypeMeta),
		Host:     address.IP,
		Ports:    ports,
		Ready:    ready,
		NodeName: address.NodeName,
	}
}

type EndpointList struct {
	*dataselect.ListMeta
	// List of endpoints
	Endpoints []api.EndpointDetail `json:"endpoints"`
}

// toEndpointList converts array of api events to endpoint List structure
func toEndpointList(endpoints []*v1.Endpoints, cluster api.ICluster) *EndpointList {
	endpointList := EndpointList{
		Endpoints: make([]api.EndpointDetail, 0),
		ListMeta:  dataselect.NewListMeta(),
	}

	for _, endpoint := range endpoints {
		for _, subSets := range endpoint.Subsets {
			for _, address := range subSets.Addresses {
				endpointList.Endpoints = append(endpointList.Endpoints, *toEndpoint(endpoint, address, subSets.Ports, true, cluster))
			}
			for _, notReadyAddress := range subSets.NotReadyAddresses {
				endpointList.Endpoints = append(endpointList.Endpoints, *toEndpoint(endpoint, notReadyAddress, subSets.Ports, false, cluster))
			}
		}
	}

	return &endpointList
}
