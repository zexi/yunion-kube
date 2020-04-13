package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"yunion.io/x/yunion-kube/pkg/apis"

	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	EndpointManager *SEndpointManager
)

func init() {
	EndpointManager = &SEndpointManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SEndpoint{},
			"k8s_endpoint",
			"k8s_endpoints"),
	}
}

type SEndpointManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SEndpoint struct {
	model.SK8SNamespaceResourceBase
}

func (m SEndpointManager) GetRawEndpoints(cluster model.ICluster, ns string) ([]*v1.Endpoints, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.EndpointLister().Endpoints(ns).List(labels.Everything())
}

func (m SEndpointManager) GetRawEndpointsByService(cluster model.ICluster, svc *v1.Service) ([]*v1.Endpoints, error) {
	eps, err := m.GetRawEndpoints(cluster, svc.GetNamespace())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Endpoints, 0)
	for _, ip := range eps {
		if ip.Name == svc.GetName() {
			ret = append(ret, ip)
		}
	}
	return ret, nil
}

func (m SEndpointManager) GetAPIEndpointsByService(cluster model.ICluster, svc *v1.Service) ([]*apis.EndpointDetail, error) {
	eps, err := m.GetRawEndpointsByService(cluster, svc)
	if err != nil {
		return nil, err
	}
	return m.GetAPIEndpoints(cluster, eps), nil
}

func (m SEndpointManager) GetAPIEndpoints(cluster model.ICluster, eps []*v1.Endpoints) []*apis.EndpointDetail {
	ret := make([]*apis.EndpointDetail, 0)
	for _, ep := range eps {
		ret = append(ret, m.GetAPIEndpoint(cluster, ep)...)
	}
	return ret
}

func (m SEndpointManager) toEndpoint(
	cluster model.ICluster,
	ep *v1.Endpoints,
	address v1.EndpointAddress,
	ports []v1.EndpointPort,
	ready bool) *apis.EndpointDetail {
	return &apis.EndpointDetail{
		ObjectMeta: apis.NewObjectMeta(ep.ObjectMeta, cluster),
		TypeMeta:   apis.NewTypeMeta(ep.TypeMeta),
		Host:       address.IP,
		Ports:      ports,
		Ready:      ready,
		NodeName:   address.NodeName,
	}
}

func (m SEndpointManager) GetAPIEndpoint(cluster model.ICluster, ep *v1.Endpoints) []*apis.EndpointDetail {
	ret := make([]*apis.EndpointDetail, 0)
	for _, subSets := range ep.Subsets {
		for _, address := range subSets.Addresses {
			ret = append(ret, m.toEndpoint(cluster, ep, address, subSets.Ports, true))
		}
		for _, notReadyAddress := range subSets.NotReadyAddresses {
			ret = append(ret, m.toEndpoint(cluster, ep, notReadyAddress, subSets.Ports, false))
		}
	}
	return ret
}
