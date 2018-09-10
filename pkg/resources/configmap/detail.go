package configmap

import (
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type ConfigMapDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// Data contains the configuration data.
	// Each key must be a valid DNS_SUBDOMAIN with an optional leading dot.
	Data map[string]string `json:"data,omitempty"`
}

func (man *SConfigMapManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetConfigMapDetail(req.GetK8sClient(), namespace, id)
}

// GetConfigMapDetail returns detailed information about a config map
func GetConfigMapDetail(client kubernetes.Interface, namespace, name string) (*ConfigMapDetail, error) {
	log.Infof("Getting details of %s config map in %s namespace", name, namespace)

	rawConfigMap, err := client.CoreV1().ConfigMaps(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return getConfigMapDetail(rawConfigMap), nil
}

func getConfigMapDetail(rawConfigMap *v1.ConfigMap) *ConfigMapDetail {
	return &ConfigMapDetail{
		ObjectMeta: api.NewObjectMeta(rawConfigMap.ObjectMeta),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindConfigMap),
		Data:       rawConfigMap.Data,
	}
}
