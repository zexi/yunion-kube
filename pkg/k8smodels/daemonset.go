package k8smodels

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var (
	DaemonSetManager *SDaemonSetManager
	_                model.IPodOwnerModel = &SDaemonSet{}
)

func init() {
	DaemonSetManager = &SDaemonSetManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SDaemonSet{},
			"daemonset",
			"daemonsets"),
	}
	DaemonSetManager.SetVirtualObject(DaemonSetManager)
}

type SDaemonSetManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SDaemonSet struct {
	model.SK8SNamespaceResourceBase
	PodTemplateResourceBase
}

func (m SDaemonSetManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameDaemonSet,
		Object:       &apps.DaemonSet{},
		KindName:     api.KindNameDaemonSet,
	}
}

func (m SDaemonSetManager) ValidateCreateData(
	ctx *model.RequestContext,
	query *jsonutils.JSONDict,
	input *api.DaemonSetCreateInput) (*api.DaemonSetCreateInput, error) {
	if _, err := m.SK8SNamespaceResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sNamespaceResourceCreateInput); err != nil {
		return input, err
	}
	return input, nil
}

func (m SDaemonSetManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input *api.DaemonSetCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	ds := &apps.DaemonSet{
		ObjectMeta: objMeta,
		Spec:       input.DaemonSetSpec,
	}
	if _, err := CreateServiceIfNotExist(ctx, &objMeta, input.Service); err != nil {
		return nil, err
	}
	return ds, nil
}

func (obj *SDaemonSet) GetAPIObject() (*api.DaemonSet, error) {
	ds := obj.GetRawDaemonSet()
	podInfo, err := obj.GetPodInfo()
	if err != nil {
		return nil, errors.Wrap(err, "get pod info")
	}
	return &api.DaemonSet{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
		PodInfo:             *podInfo,
		ContainerImages:     common.GetContainerImages(&ds.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&ds.Spec.Template.Spec),
		Selector:            ds.Spec.Selector,
		DaemonSetStatus:     *getters.GetDaemonsetStatus(podInfo, *ds),
	}, nil
}

func (obj *SDaemonSet) GetAPIDetailObject() (*api.DaemonSetDetail, error) {
	events, err := EventManager.GetEventsByObject(obj)
	if err != nil {
		return nil, err
	}
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	return &api.DaemonSetDetail{
		DaemonSet: *apiObj,
		Events:    events,
	}, nil
}

func (obj *SDaemonSet) GetRawPods() ([]*v1.Pod, error) {
	return GetRawPodsByController(obj)
}

func (obj *SDaemonSet) GetRawDaemonSet() *apps.DaemonSet {
	return obj.GetK8SObject().(*apps.DaemonSet)
}

func (obj *SDaemonSet) GetPodInfo() (*api.PodInfo, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	ds := obj.GetRawDaemonSet()
	podInfo, err := GetPodInfo(obj, ds.Status.CurrentNumberScheduled, &ds.Status.DesiredNumberScheduled, pods)
	if err != nil {
		return nil, err
	}
	return podInfo, nil
}

func (obj *SDaemonSet) ValidateUpdateData(ctx *model.RequestContext, _ *jsonutils.JSONDict, input *api.DaemonSetUpdateInput) (*api.DaemonSetUpdateInput, error) {
	return input, nil
}

func (obj *SDaemonSet) NewK8SRawObjectForUpdate(ctx *model.RequestContext, input *api.DaemonSetUpdateInput) (runtime.Object, error) {
	newObj := obj.GetRawDaemonSet().DeepCopy()
	template := &newObj.Spec.Template
	if err := obj.UpdatePodTemplate(template, input.PodTemplateUpdateInput); err != nil {
		return nil, err
	}
	return newObj, nil
}
