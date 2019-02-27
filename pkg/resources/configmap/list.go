package configmap

import (
	"k8s.io/api/core/v1"
	client "k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type ConfigMap struct {
	api.ObjectMeta
	api.TypeMeta
}

func (c ConfigMap) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(c)
}

type ConfigMapList struct {
	*common.BaseList
	configMaps []ConfigMap
}

func (l *ConfigMapList) GetResponseData() interface{} {
	return l.configMaps
}

func (l *ConfigMapList) GetConfigMaps() []ConfigMap {
	return l.configMaps
}

func (man *SConfigMapManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetK8sClient(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SConfigMapManager) ListV2(client client.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetConfigMapList(client, cluster, nsQuery, dsQuery)
}

func (man *SConfigMapManager) GetConfigMapList(client client.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*ConfigMapList, error) {
	log.Infof("Getting list of all configmap in namespace: %v", nsQuery.ToRequestParam())
	channels := &common.ResourceChannels{
		ConfigMapList: common.GetConfigMapListChannel(client, nsQuery),
	}
	return GetConfigMapListFromChannels(channels, dsQuery, cluster)
}

func (l *ConfigMapList) Append(obj interface{}) {
	l.configMaps = append(l.configMaps, ToConfigMap(obj.(v1.ConfigMap), l.GetCluster()))
}

func GetConfigMapListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ConfigMapList, error) {
	configMaps := <-channels.ConfigMapList.List
	err := <-channels.ConfigMapList.Error
	if err != nil {
		return nil, err
	}
	configMapList := &ConfigMapList{
		BaseList:   common.NewBaseList(cluster),
		configMaps: make([]ConfigMap, 0),
	}
	err = dataselect.ToResourceList(
		configMapList,
		configMaps.Items,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return configMapList, err
}
