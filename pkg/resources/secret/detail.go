package secret

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/apis"
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SSecretManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetSecretDetail(req.GetIndexer(), req.GetCluster(), namespace, id)
}

// GetSecretDetail returns returns detailed information about a secret
func GetSecretDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*apis.SecretDetail, error) {
	log.Infof("Getting details of %s secret in %s namespace", name, namespace)

	rawSecret, err := indexer.SecretLister().Secrets(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return getSecretDetail(rawSecret, cluster), nil
}

func getSecretDetail(rawSecret *v1.Secret, cluster api.ICluster) *apis.SecretDetail {
	return &apis.SecretDetail{
		Secret: *common.ToSecret(rawSecret, cluster),
		Data:   rawSecret.Data,
	}
}
