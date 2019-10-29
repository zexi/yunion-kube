package secret

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"yunion.io/x/yunion-kube/pkg/client"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/apis"
)

// SecretSpec is a common interface for the specification of different secrets.
type SecretSpec interface {
	GetName() string
	GetType() v1.SecretType
	GetNamespace() string
	GetData() map[string][]byte
}

// ImagePullSecretSpec is a specification of an image pull secret implements SecretSpec
type ImagePullSecretSpec struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	// The value of the .dockercfg property. It must be Base64 encoded.
	Data []byte `json:"data"`
}

// GetName returns the name of the ImagePullSecret
func (spec *ImagePullSecretSpec) GetName() string {
	return spec.Name
}

// GetType returns the type of the ImagePullSecret, which is always api.SecretTypeDockercfg
func (spec *ImagePullSecretSpec) GetType() v1.SecretType {
	return v1.SecretTypeDockercfg
}

// GetNamespace returns the namespace of the ImagePullSecret
func (spec *ImagePullSecretSpec) GetNamespace() string {
	return spec.Namespace
}

// GetData returns the data the secret carries, it is a single key-value pair
func (spec *ImagePullSecretSpec) GetData() map[string][]byte {
	return map[string][]byte{v1.DockerConfigKey: spec.Data}
}

// SecretsList is a response structure for a queried secrets list.
type SecretList struct {
	*common.BaseList

	// Unordered list of Secrets.
	Secrets []apis.Secret
}

func (man *SSecretManager) List(req *common.Request) (common.ListResource, error) {
	query := req.ToQuery()
	if secType, _ := req.Query.GetString("type"); secType != "" {
		filter := query.FilterQuery
		filter.Append(dataselect.NewFilterBy(dataselect.SecretTypeProperty, secType))
	}
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), query)
}

func (man *SRegistrySecretManager) List(req *common.Request) (common.ListResource, error) {
	query := req.Query
	query.Add(
		jsonutils.NewString(string(v1.SecretTypeDockerConfigJson)),
		string(dataselect.SecretTypeProperty),
	)
	return SecretManager.List(req)
}

func (man *SSecretManager) ListV2(
	indexer *client.CacheFactory,
	cluster api.ICluster,
	namespace *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return GetSecretList(indexer, cluster, namespace, dsQuery)
}

// GetSecretList returns all secrets in the given namespace.
func GetSecretList(indexer *client.CacheFactory, cluster api.ICluster, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*SecretList, error) {
	secretList, err := indexer.SecretLister().Secrets(namespace.ToRequestParam()).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return toSecretList(secretList, dsQuery, cluster)
}

func toSecretList(secrets []*v1.Secret, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*SecretList, error) {
	secretList := &SecretList{
		BaseList: common.NewBaseList(cluster),
		Secrets:  make([]apis.Secret, 0),
	}
	err := dataselect.ToResourceList(
		secretList,
		secrets,
		dataselect.NewSecretDataCell,
		dsQuery,
	)
	return secretList, err
}

func (l *SecretList) Append(obj interface{}) {
	l.Secrets = append(l.Secrets, *toSecret(obj.(*v1.Secret), l.GetCluster()))
}

func (l *SecretList) GetResponseData() interface{} {
	return l.Secrets
}
