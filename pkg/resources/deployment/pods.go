package deployment

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
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

	podList, err := pod.ToPodListByIndexerV2(indexer, rawPods, namespace, dsQuery, cluster)
	return podList, err
}
