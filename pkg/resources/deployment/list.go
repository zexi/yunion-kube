package deployment

import (
	"yunion.io/x/log"

	"yunion.io/x/jsonutils"
	//"yunion.io/x/log"
	apps "k8s.io/api/apps/v1beta2"
	"k8s.io/api/core/v1"
	//metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// Deployment is a presentation layer view of kubernetes Deployment resource. This means
// it is Deployment plus additional augmented data we can get from other sources
// (like services that target the same pods)
type Deployment struct {
	api.ObjectMeta
	api.TypeMeta

	// Aggregate information about pods belonging to this deployment
	Pods common.PodInfo `json:"pods"`

	// Container images of the Deployment
	ContainerImages []string `json:"containerImages"`

	// Init Container images of deployment
	InitContainerImages []string `json:"initContainerImages"`
}

func (d Deployment) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(d)
}

func (man *SDeploymentManager) AllowListItems(req *common.Request) bool {
	return req.AllowListItems()
}

func (man *SDeploymentManager) List(req *common.Request) (common.ListResource, error) {
	return man.GetDeploymentList(req.GetK8sClient(), req.GetNamespaceQuery(), req.ToQuery())
}

type DeploymentList struct {
	*dataselect.ListMeta
	deployments []Deployment
	replicasets []apps.ReplicaSet
	pods        []v1.Pod
	events      []v1.Event
}

func (man *SDeploymentManager) GetDeploymentList(client client.Interface, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*DeploymentList, error) {
	log.Infof("Getting list of all deployments in the cluster")

	channels := &common.ResourceChannels{
		DeploymentList: common.GetDeploymentListChannel(client, nsQuery),
		PodList:        common.GetPodListChannel(client, nsQuery),
		EventList:      common.GetEventListChannel(client, nsQuery),
		ReplicaSetList: common.GetReplicaSetListChannel(client, nsQuery),
	}

	return GetDeploymentListFromChannels(channels, dsQuery)
}

func (l *DeploymentList) Append(obj interface{}) {
	l.deployments = append(l.deployments, ToDeployment(
		obj.(apps.Deployment),
		l.replicasets,
		l.pods,
		l.events,
	))
}

func (l *DeploymentList) GetResponseData() interface{} {
	return l.deployments
}

func ToDeployment(deployment apps.Deployment, rs []apps.ReplicaSet, pods []v1.Pod, events []v1.Event) Deployment {
	matchingPods := common.FilterDeploymentPodsByOwnerReference(deployment, rs, pods)
	podInfo := common.GetPodInfo(deployment.Status.Replicas, deployment.Spec.Replicas, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)

	return Deployment{
		ObjectMeta:          api.NewObjectMeta(deployment.ObjectMeta),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindDeployment),
		ContainerImages:     common.GetContainerImages(&deployment.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&deployment.Spec.Template.Spec),
		Pods:                podInfo,
	}
}

func GetDeploymentListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery) (*DeploymentList, error) {
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
		ListMeta:    dataselect.NewListMeta(),
		deployments: make([]Deployment, 0),
		pods:        pods.Items,
		events:      events.Items,
		replicasets: rs.Items,
	}
	err = dataselect.ToResourceList(
		deploymentList,
		deployments.Items,
		dataselect.NewNamespaceDataCell,
		dsQuery)
	return deploymentList, err
}
