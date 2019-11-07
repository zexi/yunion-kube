package service

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

var ServiceManager *SServiceManager

type SServiceManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ServiceManager = &SServiceManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("k8s_service", "k8s_services"),
	}
	resources.KindManagerMap.Register(api.KindNameService, ServiceManager)
}

func (m *SServiceManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetServiceDetail(cli, cluster, namespace, name, dataselect.DefaultDataSelect())
}
