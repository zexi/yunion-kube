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

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
)

var (
	DaemonSetManager *SDaemonSetManager
	_                IClusterModel = new(SDaemonSet)
)

func init() {
	DaemonSetManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SDaemonSetManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SDaemonSet),
				"daemonsets_tbl",
				"daemonsets",
				"daemonset",
				api.ResourceNameDaemonSet,
				api.KindNameDaemonSet,
				new(apps.DaemonSet),
			),
		}
	}).(*SDaemonSetManager)
}

type SDaemonSetManager struct {
	SNamespaceResourceBaseManager
}

type SDaemonSet struct {
	SNamespaceResourceBase
}

func (m *SDaemonSetManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.DaemonSetCreateInputV2)
	data.Unmarshal(input)
	objMeta := input.ToObjectMeta()
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	ds := &apps.DaemonSet{
		ObjectMeta: objMeta,
		Spec:       input.DaemonSetSpec,
	}
	if _, err := CreateServiceIfNotExist(cli, &objMeta, input.Service); err != nil {
		return nil, err
	}
	return ds, nil
}

func (obj *SDaemonSet) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.DaemonSetDetailV2, error) {
	return api.DaemonSetDetailV2{}, nil
}

func (obj *SDaemonSet) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	return GetRawPodsByController(cli, rawObj.(metav1.Object))
}

func (obj *SDaemonSet) GetPodInfo(cli *client.ClusterManager, ds *apps.DaemonSet) (*api.PodInfo, error) {
	pods, err := obj.GetRawPods(cli, ds)
	if err != nil {
		return nil, err
	}
	podInfo, err := GetPodInfo(ds.Status.CurrentNumberScheduled, &ds.Status.DesiredNumberScheduled, pods)
	if err != nil {
		return nil, err
	}
	return podInfo, nil
}

func (obj *SDaemonSet) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	ds := k8sObj.(*apps.DaemonSet)
	detail := api.DaemonSetDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		ContainerImages:         GetContainerImages(&ds.Spec.Template.Spec),
		InitContainerImages:     GetInitContainerImages(&ds.Spec.Template.Spec),
		Selector:                ds.Spec.Selector,
	}
	podInfo, err := obj.GetPodInfo(cli, ds)
	if err != nil {
		log.Errorf("Get pod info by daemonset %s error: %v", obj.GetName(), err)
	} else {
		detail.PodInfo = *podInfo
		detail.DaemonSetStatus = *getters.GetDaemonsetStatus(podInfo, *ds)
	}
	return detail
}
