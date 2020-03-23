package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"yunion.io/x/yunion-kube/pkg/apis"

	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

type SPodManager struct {
	model.SK8SNamespaceResourceBaseManager
}

var PodManager *SPodManager

func init() {
	PodManager = &SPodManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SPod{},
			"pod",
			"pods"),
	}
	PodManager.SetVirtualObject(PodManager)
}

var (
	_ model.IK8SModel = &SPod{}
)

type SPod struct {
	model.SK8SNamespaceResourceBase
}

func (m SPodManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNamePod,
		Object:       &v1.Pod{},
	}
}

func (p SPodManager) GetRawPods(cluster model.ICluster, ns string) ([]*v1.Pod, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.PodLister().Pods(ns).List(labels.Everything())
}

func (p SPodManager) GetAllRawPods(cluster model.ICluster) ([]*v1.Pod, error) {
	return p.GetRawPods(cluster, v1.NamespaceAll)
}

func (p *SPod) GetDetails() (*apis.PodDetail, error) {
	return nil, nil
}
