package k8smodels

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

type SDaemonSetManager struct {
	model.SK8SNamespaceResourceBaseManager
}

var DaemonSetManager *SDaemonSetManager

func init() {
	DaemonSetManager = &SDaemonSetManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SDaemonSet{},
			"daemonset",
			"daemonsets"),
	}
	DaemonSetManager.SetVirtualObject(DaemonSetManager)
}

var (
	_ model.IK8SModel = &SDaemonSet{}
)

type SDaemonSet struct {
	model.SK8SNamespaceResourceBase
}

func (m SDaemonSetManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameDaemonSet,
		Object:       &apps.DaemonSet{},
	}
}

func (m SDaemonSetManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input jsonutils.JSONObject) (runtime.Object, error) {
	return nil, nil
}

func (obj *SDaemonSet) GetAPIObject() (*apis.DaemonSet, error) {
	ds := obj.GetRawDaemonSet()
	podInfo, err := obj.GetPodInfo()
	if err != nil {
		return nil, errors.Wrap(err, "get pod info")
	}
	return &apis.DaemonSet{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
		PodInfo:             *podInfo,
		ContainerImages:     common.GetContainerImages(&ds.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&ds.Spec.Template.Spec),
		Selector:            ds.Spec.Selector,
		DaemonSetStatus:     *getters.GetDaemonsetStatus(podInfo, *ds),
	}, nil
}

func (obj *SDaemonSet) GetAPIDetailsObject() (*apis.DaemonSetDetails, error) {
	events, err := EventManager.GetEventsByObject(obj)
	if err != nil {
		return nil, err
	}
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	return &apis.DaemonSetDetails{
		DaemonSet: *apiObj,
		Events:    events,
	}, nil
}

func (obj *SDaemonSet) GetRawPods() ([]*v1.Pod, error) {
	pods, err := PodManager.GetRawPods(obj.GetCluster(), obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	return common.FilterPodsByControllerRef(obj.GetK8SObject().(metav1.Object), pods), nil
}

func (obj *SDaemonSet) GetRawDaemonSet() *apps.DaemonSet {
	return obj.GetK8SObject().(*apps.DaemonSet)
}

func GetPodInfo(obj model.IK8SModel, current int32, desired *int32, pods []*v1.Pod) (*apis.PodInfo, error) {
	podInfo := common.GetPodInfo(current, desired, pods)
	warnEvents, err := EventManager.GetWarningEventsByPods(obj.GetCluster(), pods)
	if err != nil {
		return nil, err
	}
	ws := make([]apis.Event, len(warnEvents))
	for i := range warnEvents {
		ws[i] = *warnEvents[i]
	}
	podInfo.Warnings = ws
	return &podInfo, nil
}

func (obj *SDaemonSet) GetPodInfo() (*apis.PodInfo, error) {
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
