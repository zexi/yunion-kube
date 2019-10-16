package deployment

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// GetDeploymentPods returns list of pods targeting deployment.
func GetDeploymentPods(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, deploymentName string) (*pod.PodList, error) {

	deployment, err := indexer.DeploymentLister().Deployments(namespace).Get(deploymentName)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(indexer, common.NewSameNamespaceQuery(namespace), selector),
	}

	rawPods := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	events, err := event.GetPodsEvents(indexer, namespace, rawPods)
	if err != nil {
		return nil, err
	}

	podList, err := pod.ToPodList(rawPods, events, dsQuery, cluster)
	return podList, err
}
