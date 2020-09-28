package models

import (
	"context"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
)

var (
	deploymentManager *SDeploymentManager
	_                 IPodOwnerModel = new(SDeployment)
)

func init() {
	GetDeploymentManager()
}

func GetDeploymentManager() *SDeploymentManager {
	if deploymentManager == nil {
		deploymentManager = NewK8sNamespaceModelManager(func() ISyncableManager {
			return &SDeploymentManager{
				SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
					SDeployment{},
					"deployments_tbl",
					"deployment",
					"deployments",
					api.ResourceNameDeployment,
					apps.GroupName,
					apps.SchemeGroupVersion.Version,
					api.KindNameDeployment,
					new(apps.Deployment),
				),
			}
		}).(*SDeploymentManager)
	}
	return deploymentManager
}

type SDeploymentManager struct {
	SNamespaceResourceBaseManager
}

type SDeployment struct {
	SNamespaceResourceBase
}

func (m *SDeploymentManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.DeploymentCreateInput) (*api.DeploymentCreateInput, error) {
	if _, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &data.NamespaceResourceCreateInput); err != nil {
		return data, err
	}
	return data, nil
}

func (m *SDeploymentManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.DeploymentCreateInput)
	data.Unmarshal(input)
	objMeta, err := input.ToObjectMeta(model.(api.INamespaceGetter))
	if err != nil {
		return nil, err
	}
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	deploy := &apps.Deployment{
		ObjectMeta: objMeta,
		Spec:       input.DeploymentSpec,
	}
	if _, err := CreateServiceIfNotExist(cli, &objMeta, input.Service); err != nil {
		return nil, errors.Wrap(err, "create service if not exists")
	}
	return deploy, nil
}

func (obj *SDeployment) NewRemoteObjectForUpdate(cli *client.ClusterManager, remoteObj interface{}, data jsonutils.JSONObject) (interface{}, error) {
	deploy := remoteObj.(*apps.Deployment)
	input := new(api.DeploymentUpdateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	if input.Replicas != nil {
		deploy.Spec.Replicas = input.Replicas
	}
	if err := UpdatePodTemplate(&deploy.Spec.Template, input.PodTemplateUpdateInput); err != nil {
		return nil, err
	}
	return deploy, nil
}

func (m *SDeploymentManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.DeploymentListInput) (*sqlchemy.SQuery, error) {
	return m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
}

func (obj *SDeployment) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DeploymentDetailV2, error) {
	return api.DeploymentDetailV2{}, nil
}

func (obj *SDeployment) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := obj.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return errors.Wrap(err, "update deployment")
	}
	return nil
}

func (obj *SDeployment) SetStatusByRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	cli, err := obj.GetClusterClient()
	if err != nil {
		return errors.Wrap(err, "get deployment cluster client")
	}
	deploy := extObj.(*apps.Deployment)
	podInfo, err := obj.GetPodInfo(cli, deploy)
	if err != nil {
		return errors.Wrap(err, "get pod info")
	}
	deployStatus := getters.GetDeploymentStatus(podInfo, *deploy)
	return obj.SetStatus(userCred, deployStatus.Status, "update from remote")
}

func (obj *SDeployment) GetRawReplicaSets(cli *client.ClusterManager, deploy *apps.Deployment) ([]*apps.ReplicaSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, errors.Wrap(err, "deploy label as selector")
	}
	return GetReplicaSetManager().GetRawReplicaSets(cli, deploy.GetNamespace(), selector)
}

func (obj *SDeployment) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	deploy := rawObj.(*apps.Deployment)
	pods, err := GetPodManager().GetRawPods(cli, deploy.GetNamespace())
	if err != nil {
		return nil, errors.Wrapf(err, "get namespace %s pods", deploy.GetNamespace())
	}
	rss, err := obj.GetRawReplicaSets(cli, deploy)
	if err != nil {
		return nil, errors.Wrap(err, "get replicasets")
	}
	pods = FilterDeploymentPodsByOwnerReference(deploy, rss, pods)
	return pods, nil
}

func (obj *SDeployment) GetRawDeployment() (*apps.Deployment, error) {
	kObj, err := GetK8sObject(obj)
	if err != nil {
		return nil, err
	}
	return kObj.(*apps.Deployment), nil
}

func (obj *SDeployment) GetRawServices(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Service, error) {
	deploy := rawObj.(*apps.Deployment)
	return GetServiceManager().GetRawServicesByMatchLabels(cli, deploy.GetNamespace(), deploy.Spec.Selector.MatchLabels)
}

func (obj *SDeployment) GetPodInfo(cli *client.ClusterManager, deploy *apps.Deployment) (*api.PodInfo, error) {
	// TODO: refactor this code to interface
	pods, err := obj.GetRawPods(cli, deploy)
	if err != nil {
		return nil, errors.Wrap(err, "replicaset get raw pods")
	}
	return GetPodInfo(deploy.Status.Replicas, deploy.Spec.Replicas, pods)
}

func (obj *SDeployment) FindOldReplicaSets(deploy *apps.Deployment, rss []*apps.ReplicaSet) ([]*apps.ReplicaSet, []*apps.ReplicaSet, error) {
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

func (obj *SDeployment) FindNewReplicaSet(deploy *apps.Deployment) (*apps.ReplicaSet, error) {
	cli, err := obj.GetClusterClient()
	if err != nil {
		return nil, err
	}
	rss, err := obj.GetRawReplicaSets(cli, deploy)
	if err != nil {
		return nil, err
	}
	rs, err := FindNewReplicaSet(deploy, rss)
	if err != nil {
		return nil, err
	}
	return rs, nil
}

func (obj *SDeployment) GetDetails(
	cli *client.ClusterManager,
	base interface{},
	k8sObj runtime.Object,
	isList bool,
) interface{} {
	deploy := k8sObj.(*apps.Deployment)
	detail := api.DeploymentDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Replicas:                deploy.Spec.Replicas,
		ContainerImages:         GetContainerImages(&deploy.Spec.Template.Spec),
		InitContainerImages:     GetInitContainerImages(&deploy.Spec.Template.Spec),
		Selector:                deploy.Spec.Selector.MatchLabels,
	}
	podInfo, err := obj.GetPodInfo(cli, deploy)
	if err != nil {
		log.Errorf("Get pod info by deployment %s error: %v", obj.GetName(), err)
	} else {
		detail.Pods = *podInfo
		detail.DeploymentStatus = *getters.GetDeploymentStatus(podInfo, *deploy)
	}
	var rollingUpdateStrategy *api.RollingUpdateStrategy
	if deploy.Spec.Strategy.RollingUpdate != nil {
		rollingUpdateStrategy = &api.RollingUpdateStrategy{
			MaxSurge:       deploy.Spec.Strategy.RollingUpdate.MaxSurge,
			MaxUnavailable: deploy.Spec.Strategy.RollingUpdate.MaxUnavailable,
		}
	}
	detail.RollingUpdateStrategy = rollingUpdateStrategy
	detail.RevisionHistoryLimit = deploy.Spec.RevisionHistoryLimit
	return detail
}
