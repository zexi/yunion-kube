package k8smodels

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	StatefulSetManager *SStatefulSetManager
	_                  model.IPodOwnerModel = &SStatefulSet{}
)

func init() {
	StatefulSetManager = &SStatefulSetManager{
		model.NewK8SNamespaceResourceBaseManager(
			&SStatefulSet{},
			"statefulset",
			"statefulsets"),
	}
	StatefulSetManager.SetVirtualObject(StatefulSetManager)
}

type SStatefulSetManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SStatefulSet struct {
	model.SK8SNamespaceResourceBase
	ReplicaResourceBase
	PodTemplateResourceBase
}

func (m *SStatefulSetManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameStatefulSet,
		Object:       &apps.StatefulSet{},
		KindName:     apis.KindNameStatefulSet,
	}
}

func (m *SStatefulSetManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input apis.StatefulsetCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	input.ServiceName = objMeta.GetName()

	for i, p := range input.VolumeClaimTemplates {
		temp := p.DeepCopy()
		temp.SetNamespace(objMeta.GetNamespace())
		if len(temp.Spec.AccessModes) == 0 {
			temp.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
		}
		input.VolumeClaimTemplates[i] = *temp
	}

	if _, err := CreateServiceIfNotExist(ctx, &objMeta, input.Service); err != nil {
		return nil, err
	}
	ss := &apps.StatefulSet{
		ObjectMeta: objMeta,
		Spec:       input.StatefulSetSpec,
	}
	return ss, nil
}

func (obj *SStatefulSet) GetRawStatefulSet() *apps.StatefulSet {
	return obj.GetK8SObject().(*apps.StatefulSet)
}

func (obj *SStatefulSet) GetRawPods() ([]*v1.Pod, error) {
	ss := obj.GetRawStatefulSet()
	pods, err := PodManager.GetRawPods(obj.GetCluster(), ss.GetNamespace())
	if err != nil {
		return nil, err
	}
	return FilterPodsByControllerRef(ss, pods), nil
}

func (obj *SStatefulSet) GetPods() ([]*apis.Pod, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

func (obj *SStatefulSet) GetPodInfo() (*apis.PodInfo, error) {
	ss := obj.GetRawStatefulSet()
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	return GetPodInfo(obj, ss.Status.Replicas, ss.Spec.Replicas, pods)
}

func (obj *SStatefulSet) GetAPIObject() (*apis.StatefulSet, error) {
	ss := obj.GetRawStatefulSet()
	podInfo, err := obj.GetPodInfo()
	if err != nil {
		return nil, err
	}
	return &apis.StatefulSet{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
		ContainerImages:     GetContainerImages(&ss.Spec.Template.Spec),
		Replicas:            ss.Spec.Replicas,
		InitContainerImages: GetContainerImages(&ss.Spec.Template.Spec),
		Pods:                *podInfo,
		StatefulSetStatus:   *getters.GetStatefulSetStatus(podInfo, *ss),
		Selector:            ss.Spec.Selector.MatchLabels,
	}, nil
}

func (obj *SStatefulSet) GetEvents() ([]*apis.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SStatefulSet) GetRawServices() ([]*v1.Service, error) {
	ss := obj.GetRawStatefulSet()
	return ServiceManager.GetRawServicesByMatchLabels(obj.GetCluster(), obj.GetNamespace(), ss.Spec.Selector.MatchLabels)
}

func (obj *SStatefulSet) GetServices() ([]*apis.Service, error) {
	svcs, err := obj.GetRawServices()
	if err != nil {
		return nil, err
	}
	return ServiceManager.GetAPIServices(obj.GetCluster(), svcs)
}

func (obj *SStatefulSet) GetAPIDetailObject() (*apis.StatefulSetDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	pods, err := obj.GetPods()
	if err != nil {
		return nil, err
	}
	svcs, err := obj.GetServices()
	if err != nil {
		return nil, err
	}
	return &apis.StatefulSetDetail{
		StatefulSet: *apiObj,
		PodList:     pods,
		Events:      events,
		Services:    svcs,
	}, nil
}

func (obj *SStatefulSet) ValidateUpdateData(ctx *model.RequestContext, _ *jsonutils.JSONDict, input *apis.StatefulsetUpdateInput) (*apis.StatefulsetUpdateInput, error) {
	if err := obj.ReplicaResourceBase.ValidateUpdateData(input.Replicas); err != nil {
		return nil, err
	}
	return input, nil
}

func (obj *SStatefulSet) NewK8SRawObjectForUpdate(ctx *model.RequestContext, input *apis.StatefulsetUpdateInput) (*apps.StatefulSet, error) {
	newObj := obj.GetRawStatefulSet().DeepCopy()
	if input.Replicas != nil {
		newObj.Spec.Replicas = input.Replicas
	}
	template := &newObj.Spec.Template
	if err := obj.UpdatePodTemplate(template, input.PodTemplateUpdateInput); err != nil {
		return nil, err
	}
	return newObj, nil
}
