package k8smodels

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	DeploymentManager *SDeploymentManager
	_                 model.IPodOwnerModel = &SDeployment{}
)

func init() {
	DeploymentManager = &SDeploymentManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SDeployment{},
			"deployment",
			"deployments"),
	}
	DeploymentManager.SetVirtualObject(DeploymentManager)
}

type SDeploymentManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SDeployment struct {
	model.SK8SNamespaceResourceBase
	ReplicaResourceBase
	PodTemplateResourceBase
}

func (_ SDeploymentManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameDeployment,
		KindName:     apis.KindNameDeployment,
		Object:       &apps.Deployment{},
	}
}

func (m SDeploymentManager) ValidateCreateData(
	ctx *model.RequestContext,
	query *jsonutils.JSONDict,
	input *apis.DeploymentCreateInput) (*apis.DeploymentCreateInput, error) {
	if _, err := m.SK8SNamespaceResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sNamespaceResourceCreateInput); err != nil {
		return input, err
	}
	return input, nil
}

func (m SDeploymentManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input apis.DeploymentCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	deploy := &apps.Deployment{
		ObjectMeta: objMeta,
		Spec:       input.DeploymentSpec,
	}
	if _, err := CreateServiceIfNotExist(ctx, &objMeta, input.Service); err != nil {
		return nil, err
	}
	return deploy, nil
}

func (obj *SDeployment) GetRawDeployment() *apps.Deployment {
	return obj.GetK8SObject().(*apps.Deployment)
}

func (obj *SDeployment) GetRawPods() ([]*v1.Pod, error) {
	pods, err := PodManager.GetRawPods(obj.GetCluster(), obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	rss, err := obj.GetRawReplicaSets()
	if err != nil {
		return nil, err
	}
	pods = FilterDeploymentPodsByOwnerReference(obj.GetRawDeployment(), rss, pods)
	return pods, nil
}

func (obj *SDeployment) GetPods() ([]*apis.Pod, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

func (obj *SDeployment) GetPodInfo() (*apis.PodInfo, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	deploy := obj.GetRawDeployment()
	return GetPodInfo(obj, deploy.Status.Replicas, deploy.Spec.Replicas, pods)
}

func (obj *SDeployment) GetAPIObject() (*apis.Deployment, error) {
	podInfo, err := obj.GetPodInfo()
	if err != nil {
		return nil, err
	}
	deploy := obj.GetRawDeployment()
	return &apis.Deployment{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
		Pods:                *podInfo,
		Replicas:            deploy.Spec.Replicas,
		ContainerImages:     GetContainerImages(&deploy.Spec.Template.Spec),
		InitContainerImages: GetInitContainerImages(&deploy.Spec.Template.Spec),
		Selector:            deploy.Spec.Selector.MatchLabels,
		DeploymentStatus:    *getters.GetDeploymentStatus(podInfo, *deploy),
	}, nil
}

func (obj *SDeployment) GetEvents() ([]*apis.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SDeployment) GetServices() ([]*apis.Service, error) {
	deploy := obj.GetRawDeployment()
	svcs, err := ServiceManager.GetRawServicesByMatchLabels(obj.GetCluster(), obj.GetNamespace(), deploy.Spec.Selector.MatchLabels)
	if err != nil {
		return nil, err
	}
	return ServiceManager.GetAPIServices(obj.GetCluster(), svcs)
}

func (obj *SDeployment) GetStatusInfo(status *apps.DeploymentStatus) apis.StatusInfo {
	return apis.StatusInfo{
		Replicas:    status.Replicas,
		Updated:     status.UpdatedReplicas,
		Available:   status.AvailableReplicas,
		Unavailable: status.UnavailableReplicas,
	}
}

func (obj *SDeploymentManager) FindOldReplicaSets(deploy *apps.Deployment, rss []*apps.ReplicaSet) (
	[]*apps.ReplicaSet, []*apps.ReplicaSet, error) {
	var requiredRSs []*apps.ReplicaSet
	var allRSs []*apps.ReplicaSet
	newRS, err := FindNewReplicaSet(deploy, rss)
	if err != nil {
		return nil, nil, err
	}
	for _, rs := range rss {
		// Filter out new replica set
		if newRS != nil && rs.UID == newRS.UID {
			continue
		}
		allRSs = append(allRSs, rs)
		if *(rs.Spec.Replicas) != 0 {
			requiredRSs = append(requiredRSs, rs)
		}
	}
	return requiredRSs, allRSs, nil
}

func (obj *SDeployment) GetRawReplicaSets() ([]*apps.ReplicaSet, error) {
	deploy := obj.GetRawDeployment()
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, err
	}
	return ReplicaSetManager.GetRawReplicaSets(
		obj.GetCluster(),
		obj.GetNamespace(),
		selector)
}

func (obj *SDeployment) GetOldReplicaSets() ([]*apis.ReplicaSet, error) {
	deploy := obj.GetRawDeployment()
	rss, err := obj.GetRawReplicaSets()
	if err != nil {
		return nil, err
	}
	oldRs, _, err := DeploymentManager.FindOldReplicaSets(deploy, rss)
	if err != nil {
		return nil, err
	}
	return ReplicaSetManager.GetAPIReplicaSets(obj.GetCluster(), oldRs)
}

func (obj *SDeployment) GetNewReplicaSet() (*apis.ReplicaSet, error) {
	rss, err := obj.GetRawReplicaSets()
	if err != nil {
		return nil, err
	}
	rs, err := FindNewReplicaSet(obj.GetRawDeployment(), rss)
	if err != nil {
		return nil, err
	}
	if rs == nil {
		return nil, nil
	}
	return ReplicaSetManager.GetAPIReplicaSet(obj.GetCluster(), rs)
}

func (obj *SDeployment) GetAPIDetailObject() (*apis.DeploymentDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	pods, err := obj.GetPods()
	if err != nil {
		return nil, err
	}
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	services, err := obj.GetServices()
	if err != nil {
		return nil, err
	}

	deploy := obj.GetRawDeployment()
	// Extra Info
	var rollingUpdateStrategy *apis.RollingUpdateStrategy
	if deploy.Spec.Strategy.RollingUpdate != nil {
		rollingUpdateStrategy = &apis.RollingUpdateStrategy{
			MaxSurge:       deploy.Spec.Strategy.RollingUpdate.MaxSurge,
			MaxUnavailable: deploy.Spec.Strategy.RollingUpdate.MaxUnavailable,
		}
	}
	oldRss, err := obj.GetOldReplicaSets()
	if err != nil {
		return nil, err
	}
	newRs, err := obj.GetNewReplicaSet()
	if err != nil {
		return nil, err
	}
	return &apis.DeploymentDetail{
		Deployment:            *apiObj,
		Pods:                  pods,
		Services:              services,
		StatusInfo:            obj.GetStatusInfo(&deploy.Status),
		Strategy:              deploy.Spec.Strategy.Type,
		MinReadySeconds:       deploy.Spec.MinReadySeconds,
		RollingUpdateStrategy: rollingUpdateStrategy,
		OldReplicaSets:        oldRss,
		NewReplicaSet:         newRs,
		RevisionHistoryLimit:  deploy.Spec.RevisionHistoryLimit,
		Events:                events,
	}, nil
}

func (obj *SDeployment) ValidateUpdateData(ctx *model.RequestContext, _ *jsonutils.JSONDict, input *apis.DeploymentUpdateInput) (*apis.DeploymentUpdateInput, error) {
	if err := obj.ReplicaResourceBase.ValidateUpdateData(input.Replicas); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SDeployment) NewK8SRawObjectForUpdate(ctx *model.RequestContext, input *apis.DeploymentUpdateInput) (runtime.Object, error) {
	deploy := obj.GetRawDeployment().DeepCopy()
	if input.Replicas != nil {
		deploy.Spec.Replicas = input.Replicas
	}
	template := &deploy.Spec.Template
	if err := obj.UpdatePodTemplate(template, input.PodTemplateUpdateInput); err != nil {
		return nil, err
	}
	return deploy, nil
}
