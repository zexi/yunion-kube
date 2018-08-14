package deployment

import (
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	//"yunion.io/x/log"
	"k8s.io/api/core/v1"
	//metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// Deployment is a presentation layer view of kubernetes Deployment resource. This means
// it is Deployment plus additional augmented data we can get from other sources
// (like services that target the same pods)
type Deployment struct {
	ObjectMeta api.ObjectMeta `json:"objectMeta"`
	TypeMeta   api.TypeMeta   `json:"typeMeta"`

	// Aggregate information about pods belonging to this deployment
	Pods common.PodInfo `json:"pods"`

	// Container images of the Deployment
	ContainerImages []string `json:"containerImages"`

	// Init Container images of deployment
	InitContainerImages []string `json:"initContainerImages"`
}

type DeploymentList struct {
	*dataselect.ListMeta
	deployments []Deployment
}

func GetDeploymentList(client client.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*DeploymentList, error) {
	log.Infof("Getting list of all deployments in the cluster")

	channels := &common.ResourceChannels{
		DeploymentList: common.GetDeploymentListChannel(client, nsQuery),
		PodList:        common.GetPodListChannel(client, nsQuery),
		EventList:      common.GetServiceListChannel(client, nsQuery),
		ReplicaSetList: common.GetReplicaSetListChannel(client, nsQuery),
	}

	return GetDeploymentListFromChannels(channels, dsQuery)
}

func GetDeploymentListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*DeploymentList, error) {

}
