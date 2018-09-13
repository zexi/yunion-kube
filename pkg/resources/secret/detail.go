package secret

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// SecretDetail API resource provides mechanisms to inject containers with configuration data while keeping
// containers agnostic of Kubernetes
type SecretDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// Data contains the secret data.  Each key must be a valid DNS_SUBDOMAIN
	// or leading dot followed by valid DNS_SUBDOMAIN.
	// The serialized form of the secret data is a base64 encoded string,
	// representing the arbitrary (possibly non-string) data value here.
	Data map[string][]byte `json:"data"`

	// Used to facilitate programmatic handling of secret data.
	Type v1.SecretType `json:"type"`
}

func (man *SSecretManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetSecretDetail(req.GetK8sClient(), namespace, id)
}

// GetSecretDetail returns returns detailed information about a secret
func GetSecretDetail(client kubernetes.Interface, namespace, name string) (*SecretDetail, error) {
	log.Infof("Getting details of %s secret in %s namespace", name, namespace)

	rawSecret, err := client.CoreV1().Secrets(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return getSecretDetail(rawSecret), nil
}

func getSecretDetail(rawSecret *v1.Secret) *SecretDetail {
	return &SecretDetail{
		ObjectMeta: api.NewObjectMeta(rawSecret.ObjectMeta),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindSecret),
		Data:       rawSecret.Data,
		Type:       rawSecret.Type,
	}
}
