package configmap

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func ToConfigMap(configMap v1.ConfigMap) ConfigMap {
	return ConfigMap{
		ObjectMeta: api.NewObjectMeta(configMap.ObjectMeta),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindConfigMap),
	}
}
