package ingress

import (
	extensions "k8s.io/api/extensions/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

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
	return GetIngressDetail(req.GetK8sClient(), req.GetNamespaceQuery().ToRequestParam(), id)
}

// GetIngressDetail returns returns detailed information about an ingress
func GetIngressDetail(client client.Interface, namespace, name string) (*IngressDetail, error) {
	log.Infof("Getting details of %s ingress in %s namespace", name, namespace)

	rawIngress, err := client.Extensions().Ingresses(namespace).Get(name, metaV1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return getIngressDetail(rawIngress), nil
}

func getIngressDetail(rawIngress *extensions.Ingress) *IngressDetail {
	return &IngressDetail{
		ObjectMeta: api.NewObjectMeta(rawIngress.ObjectMeta),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindIngress),
		Spec:       rawIngress.Spec,
		Status:     rawIngress.Status,
	}
}
