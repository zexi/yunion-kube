package k8smodels

import (
	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ReplicaSetManager *SReplicaSetManager
	_                 model.IK8SModel = &SReplicaSet{}
)

func init() {
	ReplicaSetManager = &SReplicaSetManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SReplicaSet{},
			"replicaset",
			"replicasets"),
	}
	ReplicaSetManager.SetVirtualObject(ReplicaSetManager)
}

type SReplicaSetManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SReplicaSet struct {
	model.SK8SNamespaceResourceBase
}

func (m SReplicaSetManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameReplicaSet,
		Object:       &apps.ReplicaSet{},
	}
}

func (m SReplicaSetManager) GetRawReplicaSets(cluster model.ICluster, ns string, selector labels.Selector) ([]*apps.ReplicaSet, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.ReplicaSetLister().ReplicaSets(ns).List(selector)
}

func (m *SReplicaSetManager) GetAPIReplicaSets(cluster model.ICluster, rss []*apps.ReplicaSet) ([]*apis.ReplicaSet, error) {
	ret := make([]*apis.ReplicaSet, 0)
	err := ConvertRawToAPIObjects(m, cluster, rss, &ret)
	return ret, err
}

func (m *SReplicaSetManager) GetAPIReplicaSet(cluster model.ICluster, rs *apps.ReplicaSet) (*apis.ReplicaSet, error) {
	obj, err := model.NewK8SModelObject(m, cluster, rs)
	if err != nil {
		return nil, err
	}
	return obj.(*SReplicaSet).GetAPIObject()
}

func (obj *SReplicaSet) GetRawReplicaSet() *apps.ReplicaSet {
	return obj.GetK8SObject().(*apps.ReplicaSet)
}

func (obj *SReplicaSet) GetRawPods() ([]*v1.Pod, error) {
	return GetRawPodsByController(obj)
}

func (obj *SReplicaSet) GetPodInfo() (*apis.PodInfo, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	rs := obj.GetRawReplicaSet()
	return GetPodInfo(obj, rs.Status.Replicas, rs.Spec.Replicas, pods)
}

func (obj *SReplicaSet) GetAPIObject() (*apis.ReplicaSet, error) {
	rs := obj.GetRawReplicaSet()
	podInfo, err := obj.GetPodInfo()
	if err != nil {
		return nil, err
	}
	return &apis.ReplicaSet{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
		Pods:                *podInfo,
		ContainerImages:     GetContainerImages(&rs.Spec.Template.Spec),
		InitContainerImages: GetInitContainerImages(&rs.Spec.Template.Spec),
	}, nil
}
