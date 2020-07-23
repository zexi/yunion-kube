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
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
)

var (
	DeploymentManager *SDeploymentManager
	_                 IPodOwnerModel = new(SDeployment)
)

func init() {
	DeploymentManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SDeploymentManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SDeployment),
				"deployments_tbl",
				"deployment",
				"deployments",
				api.ResourceNameDeployment,
				api.KindNameDeployment,
				new(apps.Deployment),
			),
		}
	}).(*SDeploymentManager)
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
	objMeta := input.ToObjectMeta()
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

func (m *SDeploymentManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.DeploymentListInput) (*sqlchemy.SQuery, error) {
	return m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.NamespaceResourceListInput)
}

func (obj *SDeployment) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DeploymentDetailV2, error) {
	return api.DeploymentDetailV2{}, nil
}

func (m *SDeploymentManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []interface{} {
	return m.SNamespaceResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
}

func (obj *SDeployment) GetRawReplicaSets(cli *client.ClusterManager, deploy *apps.Deployment) ([]*apps.ReplicaSet, error) {
	selector, err := metav1.LabelSelectorAsSelector(deploy.Spec.Selector)
	if err != nil {
		return nil, errors.Wrap(err, "deploy label as selector")
	}
	return ReplicaSetManager.GetRawReplicaSets(cli, deploy.GetNamespace(), selector)
}

func (obj *SDeployment) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	deploy := rawObj.(*apps.Deployment)
	pods, err := PodManager.GetRawPods(cli, deploy.GetNamespace())
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

func (obj *SDeployment) GetPodInfo(cli *client.ClusterManager, deploy *apps.Deployment) (*api.PodInfo, error) {
	// TODO: refactor this code to interface
	pods, err := obj.GetRawPods(cli, deploy)
	if err != nil {
		return nil, errors.Wrap(err, "replicaset get raw pods")
	}
	return GetPodInfo(deploy.Status.Replicas, deploy.Spec.Replicas, pods)
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
	return detail
}
