package ingress

import (
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Ingress struct {
	api.ObjectMeta
	api.TypeMeta

	// External endpoints of this ingress.
	Endpoints []common.Endpoint `json:"endpoints"`
}

// IngressList - response structure for a queried ingress list.
type IngressList struct {
	*common.BaseList

	// Unordered list of Ingresss.
	Items []Ingress
}

func (man *SIngressManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man SIngressManager) ListV2(indexer *client.CacheFactory, cluster api.ICluster, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetIngressList(indexer, cluster, namespace, dsQuery)
}

// GetIngressList returns all ingresses in the given namespace.
func GetIngressList(indexer *client.CacheFactory, cluster api.ICluster, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*IngressList, error) {
	ingressList, err := indexer.IngressLister().Ingresses(namespace.ToRequestParam()).List(labels.Everything())

	if err != nil {
		return nil, err
	}

	return ToIngressList(ingressList, dsQuery, cluster)
}

func getEndpoints(ingress *extensions.Ingress) []common.Endpoint {
	endpoints := make([]common.Endpoint, 0)
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		for _, status := range ingress.Status.LoadBalancer.Ingress {
			endpoint := common.Endpoint{Host: status.IP}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func ToIngressList(ingresses []*extensions.Ingress, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*IngressList, error) {
	newIngressList := &IngressList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]Ingress, 0),
	}
	err := dataselect.ToResourceList(
		newIngressList,
		ingresses,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return newIngressList, err
}

func ToIngress(ingress *extensions.Ingress, cluster api.ICluster) Ingress {
	modelIngress := Ingress{
		ObjectMeta: api.NewObjectMeta(ingress.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindIngress),
		Endpoints:  getEndpoints(ingress),
	}
	return modelIngress
}

func (l *IngressList) Append(obj interface{}) {
	ingress := obj.(*extensions.Ingress)
	l.Items = append(l.Items, ToIngress(ingress, l.GetCluster()))
}

func (l *IngressList) GetResponseData() interface{} {
	return l.Items
}
