package ingress

import (
	extensions "k8s.io/api/extensions/v1beta1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// IngressDetail API resource provides mechanisms to inject containers with configuration data while keeping
// containers agnostic of Kubernetes
type IngressDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// TODO: replace this with UI specific fields.
	// Spec is the desired state of the Ingress.
	Spec extensions.IngressSpec `json:"spec"`

	// Status is the current state of the Ingress.
	Status extensions.IngressStatus `json:"status"`
}

func (man *SIngressManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetIngressDetail(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery().ToRequestParam(), id)
}

// GetIngressDetail returns returns detailed information about an ingress
func GetIngressDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*IngressDetail, error) {
	log.Infof("Getting details of %s ingress in %s namespace", name, namespace)

	rawIngress, err := indexer.IngressLister().Ingresses(namespace).Get(name)

	if err != nil {
		return nil, err
	}

	return getIngressDetail(rawIngress, cluster), nil
}

func getIngressDetail(rawIngress *extensions.Ingress, cluster api.ICluster) *IngressDetail {
	return &IngressDetail{
		ObjectMeta: api.NewObjectMetaV2(rawIngress.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindIngress),
		Spec:       rawIngress.Spec,
		Status:     rawIngress.Status,
	}
}
