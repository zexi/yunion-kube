package deployment

import (
	"reflect"

	apps "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	//hpa "yunion.io/x/yunion-kube/pkg/resources/horizontalpodautoscaler"
	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/replicaset"
	"yunion.io/x/yunion-kube/pkg/resources/service"
)

func (man *SDeploymentManager) Get(req *common.Request, id string) (interface{}, error) {
	namespace := req.GetNamespaceQuery().ToRequestParam()
	return GetDeploymentDetail(req.GetIndexer(), req.GetCluster(), namespace, id)
}

func GetDeploymentDetail(indexer *client.CacheFactory, cluster api.ICluster, namespace, deploymentName string) (*api.DeploymentDetail, error) {
	log.Infof("Getting details of %q deployment in %q namespace", deploymentName, namespace)
	deployment, err := indexer.DeploymentLister().Deployments(namespace).Get(deploymentName)
	if err != nil {
		return nil, err
	}
	selector, err := metaV1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}
	channels := &common.ResourceChannels{
		ReplicaSetList: common.GetReplicaSetListChannelWithOptions(indexer,
			common.NewSameNamespaceQuery(namespace), selector),
		PodList: common.GetPodListChannelWithOptions(indexer,
			common.NewSameNamespaceQuery(namespace), selector),
		EventList:   common.GetEventListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		ServiceList: common.GetServiceListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}

	rawRs := <-channels.ReplicaSetList.List
	err = <-channels.ReplicaSetList.Error
	if err != nil {
		return nil, err
	}

	rawPods := <-channels.PodList.List
	err = <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	rawEvents := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	svcList, err := service.GetServiceListFromChannels(channels, dataselect.DefaultDataSelect(), cluster)
	if err != nil {
		return nil, err
	}

	commonDeployment := ToDeployment(deployment, rawRs, rawPods, rawEvents, cluster)

	podList, err := GetDeploymentPods(indexer, cluster, dataselect.DefaultDataSelect(), namespace, deploymentName)
	if err != nil {
		return nil, err
	}

	eventList, err := event.GetResourceEvents(indexer, cluster, dataselect.DefaultDataSelect(), namespace, deploymentName)
	if err != nil {
		return nil, err
	}

	oldReplicaSetList, err := GetDeploymentOldReplicaSets(indexer, cluster, dataselect.DefaultDataSelect(), namespace, deploymentName)
	if err != nil {
		return nil, err
	}

	rawRepSets := make([]*apps.ReplicaSet, 0)
	for i := range rawRs {
		rawRepSets = append(rawRepSets, rawRs[i])
	}
	newRs, err := FindNewReplicaSet(deployment, rawRepSets)
	if err != nil {
		return nil, err
	}

	var newReplicaSet api.ReplicaSet
	if newRs != nil {
		matchingPods := common.FilterPodsByControllerRef(newRs, rawPods)
		newRsPodInfo := common.GetPodInfo(newRs.Status.Replicas, newRs.Spec.Replicas, matchingPods)
		events, err := event.GetPodsEvents(indexer, namespace, matchingPods)
		if err != nil {
			return nil, err
		}

		newRsPodInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)
		newReplicaSet = replicaset.ToReplicaSet(newRs, &newRsPodInfo, cluster)
	}

	// Extra Info
	var rollingUpdateStrategy *api.RollingUpdateStrategy
	if deployment.Spec.Strategy.RollingUpdate != nil {
		rollingUpdateStrategy = &api.RollingUpdateStrategy{
			MaxSurge:       deployment.Spec.Strategy.RollingUpdate.MaxSurge,
			MaxUnavailable: deployment.Spec.Strategy.RollingUpdate.MaxUnavailable,
		}
	}

	// filter services by selector
	podLabel := deployment.Spec.Selector.MatchLabels
	svcs := make([]api.Service, 0)
	for _, svc := range svcList.Services {
		if reflect.DeepEqual(svc.Selector, podLabel) {
			svcs = append(svcs, svc)
		}
	}

	return &api.DeploymentDetail{
		Deployment:            commonDeployment,
		PodList:               podList.Pods,
		ServiceList:           svcs,
		StatusInfo:            GetStatusInfo(&deployment.Status),
		Strategy:              deployment.Spec.Strategy.Type,
		MinReadySeconds:       deployment.Spec.MinReadySeconds,
		RollingUpdateStrategy: rollingUpdateStrategy,
		OldReplicaSetList:     oldReplicaSetList.ReplicaSets,
		NewReplicaSet:         newReplicaSet,
		RevisionHistoryLimit:  deployment.Spec.RevisionHistoryLimit,
		EventList:             eventList.Events,
		//HorizontalPodAutoscalerList: *hpas,
	}, nil
}

func GetStatusInfo(deploymentStatus *apps.DeploymentStatus) api.StatusInfo {
	return api.StatusInfo{
		Replicas:    deploymentStatus.Replicas,
		Updated:     deploymentStatus.UpdatedReplicas,
		Available:   deploymentStatus.AvailableReplicas,
		Unavailable: deploymentStatus.UnavailableReplicas,
	}
}
