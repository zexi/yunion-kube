package configmap

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

func ToConfigMap(configMap *v1.ConfigMap, cluster api.ICluster) api.ConfigMap {
	return api.ConfigMap{
		ObjectMeta: api.NewObjectMeta(configMap.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(configMap.TypeMeta),
	}
}
