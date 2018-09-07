package namespace

import (
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"
)

// NamespaceSpec is a specification of namespace to create.
type NamespaceSpec struct {
	// Name of the namespace.
	Name string `json:"name"`
}

// CreateNamespace creates namespace based on given specification.
func CreateNamespace(spec *NamespaceSpec, client kubernetes.Interface) error {
	log.Infof("Creating namespace %s", spec.Name)

	namespace := &api.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: spec.Name,
		},
	}

	_, err := client.CoreV1().Namespaces().Create(namespace)
	return err
}
