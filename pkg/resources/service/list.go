package service

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
)

func (man *SServiceManager) List(req *common.Request) (common.ListResource, error) {
	query := req.ToQuery()
	if svcType, _ := req.Query.GetString("type"); svcType != "" {
		filter := query.FilterQuery
		filter.Append(dataselect.NewFilterBy(dataselect.ServiceTypeProperty, svcType))
	}
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), query)
}

func (man *SServiceManager) ListV2(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetServiceList(indexer, cluster, nsQuery, dsQuery)
}

func (man *SServiceManager) GetServiceList(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*ServiceList, error) {
	log.Infof("Getting list of all services in the cluster")
	channels := &common.ResourceChannels{
		ServiceList: common.GetServiceListChannel(indexer, nsQuery),
	}

	return GetServiceListFromChannels(channels, dsQuery, cluster)
}

type ServiceList struct {
	*common.BaseList
	Services []api.Service
}

func (l *ServiceList) Append(obj interface{}) {
	l.Services = append(l.Services, ToService(obj.(*v1.Service), l.GetCluster()))
}

func (l *ServiceList) GetResponseData() interface{} {
	return l.Services
}

func GetServiceListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*ServiceList, error) {
	services := <-channels.ServiceList.List
	err := <-channels.ServiceList.Error
	if err != nil {
		return nil, err
	}

	serviceList := &ServiceList{
		BaseList: common.NewBaseList(cluster),
		Services: make([]api.Service, 0),
	}
	err = dataselect.ToResourceList(
		serviceList,
		services,
		dataselect.NewServiceDataCell,
		dsQuery)
	return serviceList, err
}
