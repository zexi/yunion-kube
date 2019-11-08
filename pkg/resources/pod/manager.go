package pod

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
)

var PodManager *SPodManager

type SPodManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	PodManager = &SPodManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("pod", "pods"),
	}
	resources.KindManagerMap.Register(api.KindNamePod, PodManager)
}

func (m *SPodManager) GetDetails(cli *client.CacheFactory, cluster api.ICluster, namespace, name string) (interface{}, error) {
	return GetPodDetail(cli, cluster, namespace, name)
}
