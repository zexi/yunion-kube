package ingress

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
)

var IngressManager *SIngressManager

type SIngressManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	IngressManager = &SIngressManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("ingress", "ingresses"),
	}
	resources.KindManagerMap.Register(api.KindNameIngress, IngressManager)
}

func (m *SIngressManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetIngressDetail(cli, cluster, namespace, name)
}
