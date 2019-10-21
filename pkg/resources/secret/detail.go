package secret

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
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
	return GetSecretDetail(req.GetIndexer(), req.GetCluster(), namespace, id)
}

// GetSecretDetail returns returns detailed information about a secret
func GetSecretDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*SecretDetail, error) {
	log.Infof("Getting details of %s secret in %s namespace", name, namespace)

	rawSecret, err := indexer.SecretLister().Secrets(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return getSecretDetail(rawSecret, cluster), nil
}

func getSecretDetail(rawSecret *v1.Secret, cluster api.ICluster) *SecretDetail {
	return &SecretDetail{
		ObjectMeta: api.NewObjectMetaV2(rawSecret.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindSecret),
		Data:       rawSecret.Data,
		Type:       rawSecret.Type,
	}
}
