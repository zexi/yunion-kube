package configmap

import (
	"k8s.io/api/core/v1"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/apis"
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

type ConfigMapList struct {
	*common.BaseList
	configMaps []apis.ConfigMap
}

func (l *ConfigMapList) GetResponseData() interface{} {
	return l.configMaps
}

func (l *ConfigMapList) GetConfigMaps() []apis.ConfigMap {
	return l.configMaps
}

func (man *SConfigMapManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SConfigMapManager) ListV2(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetConfigMapList(indexer, cluster, nsQuery, dsQuery)
}

func (man *SConfigMapManager) GetConfigMapList(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*ConfigMapList, error) {
	log.Infof("Getting list of all configmap in namespace: %v", nsQuery.ToRequestParam())
	channels := &common.ResourceChannels{
		ConfigMapList: common.GetConfigMapListChannel(indexer, nsQuery),
	}
	return GetConfigMapListFromChannels(channels, dsQuery, cluster)
}

func (l *ConfigMapList) Append(obj interface{}) {
	l.configMaps = append(l.configMaps, common.ToConfigMap(obj.(*v1.ConfigMap), l.GetCluster()))
}

func GetConfigMapListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ConfigMapList, error) {
	configMaps := <-channels.ConfigMapList.List
	err := <-channels.ConfigMapList.Error
	if err != nil {
		return nil, err
	}
	configMapList := &ConfigMapList{
		BaseList:   common.NewBaseList(cluster),
		configMaps: make([]apis.ConfigMap, 0),
	}
	err = dataselect.ToResourceList(
		configMapList,
		configMaps,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return configMapList, err
}
