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
	Pods common.PodInfo `json:"podsInfo"`

	// Container images of the Deployment
	ContainerImages []string `json:"containerImages"`

	// Init Container images of deployment
	InitContainerImages []string `json:"initContainerImages"`

	Status   string            `json:"status"`
	Selector map[string]string `json:"selector"`
}

func (d Deployment) ToListItem() jsonutils.JSONObject {
	return jsonutils.Marshal(d)
}

func (man *SDeploymentManager) List(req *common.Request) (common.ListResource, error) {
	return man.ListV2(req.GetK8sClient(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

func (man *SDeploymentManager) ListV2(client client.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error) {
	return man.GetDeploymentList(client, cluster, nsQuery, dsQuery)
}

type DeploymentList struct {
	*common.BaseList
	deployments []Deployment
	replicasets []apps.ReplicaSet
	pods        []v1.Pod
	events      []v1.Event
}

func (l *DeploymentList) GetDeployments() []Deployment {
	return l.deployments
}

func (man *SDeploymentManager) GetDeploymentList(client client.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*DeploymentList, error) {
	log.Infof("Getting list of all deployments in the cluster")

	channels := &common.ResourceChannels{
		DeploymentList: common.GetDeploymentListChannel(client, nsQuery),
		PodList:        common.GetPodListChannel(client, nsQuery),
		EventList:      common.GetEventListChannel(client, nsQuery),
		ReplicaSetList: common.GetReplicaSetListChannel(client, nsQuery),
	}

	return GetDeploymentListFromChannels(channels, dsQuery, cluster)
}

func (l *DeploymentList) Append(obj interface{}) {
	l.deployments = append(l.deployments, ToDeployment(
		obj.(apps.Deployment),
		l.replicasets,
		l.pods,
		l.events,
		l.GetCluster(),
	))
}

func (l *DeploymentList) GetResponseData() interface{} {
	return l.deployments
}

func ToDeployment(deployment apps.Deployment, rs []apps.ReplicaSet, pods []v1.Pod, events []v1.Event, cluster api.ICluster) Deployment {
	matchingPods := common.FilterDeploymentPodsByOwnerReference(deployment, rs, pods)
	podInfo := common.GetPodInfo(deployment.Status.Replicas, deployment.Spec.Replicas, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(events, matchingPods)

	return Deployment{
		ObjectMeta:          api.NewObjectMetaV2(deployment.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindDeployment),
		ContainerImages:     common.GetContainerImages(&deployment.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&deployment.Spec.Template.Spec),
		Pods:                podInfo,
		Status:              podInfo.GetStatus(),
		Selector:            deployment.Spec.Selector.MatchLabels,
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
