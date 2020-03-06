package deployment

import (
	apps "k8s.io/api/apps/v1beta2"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/replicaset"
)

//GetDeploymentOldReplicaSets returns old replica sets targeting Deployment with given name
func GetDeploymentOldReplicaSets(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery,
	namespace string, deploymentName string) (*replicaset.ReplicaSetList, error) {

	oldReplicaSetList := &replicaset.ReplicaSetList{
		BaseList:    common.NewBaseList(cluster),
		ReplicaSets: make([]api.ReplicaSet, 0),
	}

	deployment, err := indexer.DeploymentLister().Deployments(namespace).Get(deploymentName)
	if err != nil {
		return oldReplicaSetList, err
	}

	selector, err := metaV1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return oldReplicaSetList, err
	}

	channels := &common.ResourceChannels{
		ReplicaSetList: common.GetReplicaSetListChannelWithOptions(indexer,
			common.NewSameNamespaceQuery(namespace), selector),
		PodList: common.GetPodListChannelWithOptions(indexer,
			common.NewSameNamespaceQuery(namespace), selector),
		EventList: common.GetEventListChannelWithOptions(indexer,
			common.NewSameNamespaceQuery(namespace), selector),
	}

	rawRs := <-channels.ReplicaSetList.List
	if err := <-channels.ReplicaSetList.Error; err != nil {
		return oldReplicaSetList, err
	}

	rawPods := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return oldReplicaSetList, err
	}

	rawEvents := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	rawRepSets := make([]*apps.ReplicaSet, 0)
	for i := range rawRs {
		rawRepSets = append(rawRepSets, rawRs[i])
	}
	oldRs, _, err := FindOldReplicaSets(deployment, rawRepSets)
	if err != nil {
		return oldReplicaSetList, err
	}

	oldReplicaSets := make([]*apps.ReplicaSet, len(oldRs))
	for i, replicaSet := range oldRs {
		oldReplicaSets[i] = replicaSet
	}

	oldReplicaSetList, err = replicaset.ToReplicaSetList(oldReplicaSets, rawPods, rawEvents, dsQuery, cluster)
	return oldReplicaSetList, err
}
