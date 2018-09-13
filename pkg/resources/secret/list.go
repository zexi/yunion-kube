package secret

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
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

// Secret is a single secret returned to the frontend.
type Secret struct {
	api.ObjectMeta
	api.TypeMeta
	Type v1.SecretType `json:"type"`
}

// SecretsList is a response structure for a queried secrets list.
type SecretList struct {
	*dataselect.ListMeta

	// Unordered list of Secrets.
	Secrets []Secret
}

func (man *SSecretManager) AllowListItems(req *common.Request) bool {
	return req.AllowListItems()
}

func (man *SSecretManager) List(req *common.Request) (common.ListResource, error) {
	return GetSecretList(req.GetK8sClient(), req.GetNamespaceQuery(), req.ToQuery())
}

// GetSecretList returns all secrets in the given namespace.
func GetSecretList(client kubernetes.Interface, namespace *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*SecretList, error) {
	secretList, err := client.CoreV1().Secrets(namespace.ToRequestParam()).List(api.ListEverything)
	if err != nil {
		return nil, err
	}

	return toSecretList(secretList.Items, dsQuery)
}

func toSecretList(secrets []v1.Secret, dsQuery *dataselect.DataSelectQuery) (*SecretList, error) {
	secretList := &SecretList{
		ListMeta: dataselect.NewListMeta(),
		Secrets:  make([]Secret, 0),
	}
	err := dataselect.ToResourceList(
		secretList,
		secrets,
		dataselect.NewNamespaceDataCell,
		dsQuery,
	)
	return secretList, err
}

func (l *SecretList) Append(obj interface{}) {
	l.Secrets = append(l.Secrets, *toSecret(obj.(v1.Secret)))
}

func (l *SecretList) GetResponseData() interface{} {
	return l.Secrets
}
