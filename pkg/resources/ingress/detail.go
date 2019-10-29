package ingress

import (
	extensions "k8s.io/api/extensions/v1beta1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SIngressManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetIngressDetail(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery().ToRequestParam(), id)
}

// GetIngressDetail returns returns detailed information about an ingress
func GetIngressDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*api.IngressDetail, error) {
	log.Infof("Getting details of %s ingress in %s namespace", name, namespace)

	rawIngress, err := indexer.IngressLister().Ingresses(namespace).Get(name)

	if err != nil {
		return nil, err
	}

	return getIngressDetail(rawIngress, cluster), nil
}

func getIngressDetail(rawIngress *extensions.Ingress, cluster api.ICluster) *api.IngressDetail {
	return &api.IngressDetail{
		Ingress: ToIngress(rawIngress, cluster),
		Spec:       rawIngress.Spec,
		Status:     rawIngress.Status,
	}
}
