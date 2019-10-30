package secret

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/apis"
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

// CreateSecret creates a single secret using the cluster API client
func CreateSecret(client kubernetes.Interface, cluster api.ICluster, spec SecretSpec) (*apis.Secret, error) {
	namespace := spec.GetNamespace()
	secret := &v1.Secret{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      spec.GetName(),
			Namespace: namespace,
		},
		Type: spec.GetType(),
		Data: spec.GetData(),
	}
	_, err := client.CoreV1().Secrets(namespace).Create(secret)
	return common.ToSecret(secret, cluster), err
}
