package deployment

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

func (man *SDeploymentManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SDeploymentManager) ListV2(client *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetDeploymentList(client, cluster, nsQuery, dsQuery)
}

type DeploymentList struct {
	*common.BaseList
	deployments []api.Deployment
	replicasets []*apps.ReplicaSet
	pods        []*v1.Pod
	events      []*v1.Event
}

func (l *DeploymentList) GetDeployments() []api.Deployment {
	return l.deployments
}

func (man *SDeploymentManager) GetDeploymentList(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*DeploymentList, error) {
	log.Infof("Getting list of all deployments in the cluster")

	channels := &common.ResourceChannels{
		DeploymentList: common.GetDeploymentListChannel(indexer, nsQuery),
		PodList:        common.GetPodListChannel(indexer, nsQuery),
		EventList:      common.GetEventListChannel(indexer, nsQuery),
		ReplicaSetList: common.GetReplicaSetListChannel(indexer, nsQuery),
	}

	return GetDeploymentListFromChannels(channels, dsQuery, cluster)
}

func (l *DeploymentList) Append(obj interface{}) {
	l.deployments = append(l.deployments, ToDeployment(
		obj.(*apps.Deployment),
		l.replicasets,
		l.pods,
		l.events,
		l.GetCluster(),
	))
}

func (l *DeploymentList) GetResponseData() interface{} {
	return l.deployments
}

func ToDeployment(deployment *apps.Deployment, rs []*apps.ReplicaSet, pods []*v1.Pod, events []*v1.Event, cluster api.ICluster) api.Deployment {
	matchingPods := common.FilterDeploymentPodsByOwnerReference(deployment, rs, pods)
	podInfo := common.GetPodInfo(deployment.Status.Replicas, deployment.Spec.Replicas, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)

	return api.Deployment{
		ObjectMeta:          api.NewObjectMeta(deployment.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(deployment.TypeMeta),
		ContainerImages:     common.GetContainerImages(&deployment.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&deployment.Spec.Template.Spec),
		Pods:                podInfo,
		DeploymentStatus:    *getters.GetDeploymentStatus(&podInfo, *deployment),
		Selector:            deployment.Spec.Selector.MatchLabels,
		Replicas:            deployment.Spec.Replicas,
	}
}

func GetDeploymentListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*DeploymentList, error) {
	deployments := <-channels.DeploymentList.List
	err := <-channels.DeploymentList.Error
	if err != nil {
		return nil, err
	}

	pods := <-channels.PodList.List
	err = <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	events := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	rs := <-channels.ReplicaSetList.List
	err = <-channels.ReplicaSetList.Error
	if err != nil {
		return nil, err
	}

	deploymentList := &DeploymentList{
		BaseList:    common.NewBaseList(cluster),
		deployments: make([]api.Deployment, 0),
		pods:        pods,
		events:      events,
		replicasets: rs,
	}
	err = dataselect.ToResourceList(
		deploymentList,
		deployments,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return deploymentList, err
}
