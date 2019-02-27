package deployment

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// GetDeploymentPods returns list of pods targeting deployment.
func GetDeploymentPods(client client.Interface, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, deploymentName string) (*pod.PodList, error) {

	deployment, err := client.AppsV1beta2().Deployments(namespace).Get(deploymentName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PodList:        common.GetPodListChannel(client, common.NewSameNamespaceQuery(namespace)),
		ReplicaSetList: common.GetReplicaSetListChannel(client, common.NewSameNamespaceQuery(namespace)),
	}

	rawPods := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	rawRs := <-channels.ReplicaSetList.List
	err = <-channels.ReplicaSetList.Error
	if err != nil {
		return nil, err
	}

	pods := common.FilterDeploymentPodsByOwnerReference(*deployment, rawRs.Items, rawPods.Items)
	events, err := event.GetPodsEvents(client, namespace, pods)
	if err != nil {
		return nil, err
	}

	podList, err := pod.ToPodList(pods, events, dsQuery, cluster)
	return podList, err
}
