package configmap

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func ToConfigMap(configMap v1.ConfigMap, cluster api.ICluster) ConfigMap {
	return ConfigMap{
		ObjectMeta: api.NewObjectMetaV2(configMap.ObjectMeta, cluster),
		TypeMeta:   api.NewTypeMeta(api.ResourceKindConfigMap),
	}
}
