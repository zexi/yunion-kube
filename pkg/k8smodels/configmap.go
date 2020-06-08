package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ConfigMapManager *SConfigMapManager
	_                model.IPodOwnerModel = &SConfigMap{}
)

func init() {
	ConfigMapManager = &SConfigMapManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SConfigMap), "configmap", "configmaps"),
	}
	ConfigMapManager.SetVirtualObject(ConfigMapManager)
}

type SConfigMapManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SConfigMap struct {
	model.SK8SNamespaceResourceBase
}

func (m SConfigMapManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameConfigMap,
		Object:       &v1.ConfigMap{},
		KindName:     api.KindNameConfigMap,
	}
}

func (m SConfigMapManager) ValidateCreateData(
	ctx *model.RequestContext, query *jsonutils.JSONDict,
	input *api.ConfigMapCreateInput) (*api.ConfigMapCreateInput, error) {
	if len(input.Data) == 0 {
		return nil, httperrors.NewNotAcceptableError("data is empty")
	}
	return input, nil
}

func (m SConfigMapManager) NewK8SRawObjectForCreate(_ *model.RequestContext, input api.ConfigMapCreateInput) (runtime.Object, error) {
	return &v1.ConfigMap{
		ObjectMeta: input.ToObjectMeta(),
		Data:       input.Data,
	}, nil
}

func (m SConfigMapManager) GetRawConfigMaps(cluster model.ICluster, ns string) ([]*v1.ConfigMap, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.ConfigMapLister().ConfigMaps(ns).List(labels.Everything())
}

func (m *SConfigMapManager) GetAPIConfigMaps(cluster model.ICluster, cfgs []*v1.ConfigMap) ([]*api.ConfigMap, error) {
	ret := make([]*api.ConfigMap, 0)
	err := ConvertRawToAPIObjects(m, cluster, cfgs, &ret)
	return ret, err
}

func (m SConfigMap) GetRawConfigMap() *v1.ConfigMap {
	return m.GetK8SObject().(*v1.ConfigMap)
}

func (m SConfigMap) GetAPIObject() (*api.ConfigMap, error) {
	return &api.ConfigMap{
		ObjectMeta: m.GetObjectMeta(),
		TypeMeta:   m.GetTypeMeta(),
	}, nil
}

func (m SConfigMap) getMountRawPods() ([]*v1.Pod, error) {
	cfgName := m.GetName()
	ns := m.GetNamespace()
	rawPods, err := PodManager.GetRawPods(m.GetCluster(), ns)
	if err != nil {
		return nil, err
	}
	mountPods := make([]*v1.Pod, 0)
	markMap := make(map[string]bool, 0)
	for _, pod := range rawPods {
		cfgs := GetPodConfigMapVolumes(pod)
		for _, cfg := range cfgs {
			if cfg.ConfigMap.Name == cfgName {
				if _, ok := markMap[pod.GetName()]; !ok {
					mountPods = append(mountPods, pod)
					markMap[pod.GetName()] = true
				}
			}
		}
	}
	return mountPods, err
}

func (m SConfigMap) GetRawPods() ([]*v1.Pod, error) {
	return m.getMountRawPods()
}

func (m SConfigMap) GetMountPods() ([]*api.Pod, error) {
	mountPods, err := m.getMountRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(m.GetCluster(), mountPods)
}

func (m SConfigMap) GetAPIDetailObject() (*api.ConfigMapDetail, error) {
	conf, err := m.GetAPIObject()
	if err != nil {
		return nil, err
	}
	mntPods, err := m.GetMountPods()
	if err != nil {
		return nil, err
	}
	rawConf := m.GetRawConfigMap()
	return &api.ConfigMapDetail{
		ConfigMap: *conf,
		Data:      rawConf.Data,
		Pods:      mntPods,
	}, nil
}

func (m SConfigMap) NewK8SRawObjectForUpdate(ctx *model.RequestContext, input api.ConfigMapUpdateInput) (runtime.Object, error) {
	updateObj := m.GetRawConfigMap().DeepCopy()
	for k, v := range input.Data {
		updateObj.Data[k] = v
	}
	return updateObj, nil
}
