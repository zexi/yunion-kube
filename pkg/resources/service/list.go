package service

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Service struct {
	api.ObjectMeta
	api.TypeMeta

	// InternalEndpoint of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	InternalEndpoint common.Endpoint `json:"internalEndpoint"`

	// ExternalEndpoints of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	ExternalEndpoints []common.Endpoint `json:"externalEndpoint"`

	// Label selector of the service
	Selector map[string]string `json:"selector"`

	// Type determines how the service will be exposed. Valid options: ClusterIP, NodePort, LoadBalancer
	Type v1.ServiceType `json:"type"`

	// ClusterIP is usually assigned by the master. Valid values are None, empty string (""), or
	// a valid IP address. None can be specified for headless services when proxying is not required
	ClusterIP string `json:"clusterIP"`
}

// ToListItem dynamic called by common.ToListJsonData
func (s Service) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(s)
}

func (man *SServiceManager) List(req *common.Request) (common.ListResource, error) {
	query := req.ToQuery()
	if svcType, _ := req.Query.GetString("type"); svcType != "" {
		filter := query.FilterQuery
		filter.Append(dataselect.NewFilterBy(dataselect.ServiceTypeProperty, svcType))
	}
	return man.ListV2(req.GetK8sClient(), req.GetCluster(), req.GetNamespaceQuery(), query)
}

func (man *SServiceManager) ListV2(client kubernetes.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetServiceList(client, cluster, nsQuery, dsQuery)
}

func (man *SServiceManager) GetServiceList(client kubernetes.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*ServiceList, error) {
	log.Infof("Getting list of all services in the cluster")
	channels := &common.ResourceChannels{
		ServiceList: common.GetServiceListChannel(client, nsQuery),
	}

	return GetServiceListFromChannels(channels, dsQuery, cluster)
}

type ServiceList struct {
	*common.BaseList
	Services []Service
}

func (l *ServiceList) Append(obj interface{}) {
	l.Services = append(l.Services, ToService(obj.(v1.Service), l.GetCluster()))
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
		Services: make([]Service, 0),
	}
	err = dataselect.ToResourceList(
		serviceList,
		services.Items,
		dataselect.NewServiceDataCell,
		dsQuery)
	return serviceList, err
}
