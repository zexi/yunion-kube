package configmap

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/apis"
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
)

func (man *SConfigMapManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetConfigMapDetail(req.GetIndexer(), req.GetCluster(), namespace, id)
}

// GetConfigMapDetail returns detailed information about a config map
func GetConfigMapDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, name string) (*apis.ConfigMapDetail, error) {
	log.Infof("Getting details of %s config map in %s namespace", name, namespace)

	rawConfigMap, err := indexer.ConfigMapLister().ConfigMaps(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	pods, err := indexer.PodLister().Pods(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return getConfigMapDetail(indexer, rawConfigMap, pods, cluster)
}

func getConfigMapDetail(indexer *client.CacheFactory, rawConfigMap *v1.ConfigMap, pods []*v1.Pod, cluster api.ICluster) (*apis.ConfigMapDetail, error) {
	pods = getMountPods(rawConfigMap.GetName(), pods)
	mountPods, err := pod.ToPodListByIndexer(indexer, rawConfigMap.Namespace, dataselect.DefaultDataSelect(), labels.Everything(), cluster)
	if err != nil {
		return nil, err
	}
	return &apis.ConfigMapDetail{
		ConfigMap: ToConfigMap(rawConfigMap, cluster),
		Data:      rawConfigMap.Data,
		Pods:      mountPods.GetPods(),
	}, nil
}

func getPodConfigMaps(vols []v1.Volume) []v1.Volume {
	var cfgs []v1.Volume
	for _, vol := range vols {
		if vol.VolumeSource.ConfigMap != nil {
			cfgs = append(cfgs, vol)
		}
	}
	return cfgs
}

func getMountPods(cfgName string, pods []*v1.Pod) []*v1.Pod {
	ret := []*v1.Pod{}
	for _, pod := range pods {
		cfgs := getPodConfigMaps(pod.Spec.Volumes)
		for _, cfg := range cfgs {
			if cfg.ConfigMap.Name == cfgName {
				ret = append(ret, pod)
			}
		}
	}
	return ret
}
