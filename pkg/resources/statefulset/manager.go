package statefulset

import (
	apps "k8s.io/api/apps/v1"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var StatefulSetManager *SStatefuleSetManager

type SStatefuleSetManager struct {
	*resources.SNamespaceResourceManager
}

func (m *SStatefuleSetManager) IsRawResource() bool {
	return false
}

func init() {
	StatefulSetManager = &SStatefuleSetManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("statefulset", "statefulsets"),
	}
	resources.KindManagerMap.Register(apis.KindNameStatefulSet, StatefulSetManager)
}

func (m *SStatefuleSetManager) GetDetails(cli *client.CacheFactory, cluster apis.ICluster, namespace, name string) (interface{}, error) {
	return GetStatefulSetDetail(cli, cluster, namespace, name)
}

func (m *SStatefuleSetManager) get(req *common.Request, id string) (*apps.StatefulSet, error) {
	cli := req.GetK8sManager()
	namespace := req.GetDefaultNamespace()
	indexer := cli.GetIndexer()
	return indexer.StatefulSetLister().StatefulSets(namespace).Get(id)
}

func (m *SStatefuleSetManager) AllowUpdateItem(req *common.Request, id string) bool {
	return m.SNamespaceResourceManager.AllowUpdateItem(req, id)
}

func (m *SStatefuleSetManager) Update(req *common.Request, id string) (interface{}, error) {
	deploy, err := m.get(req, id)
	if err != nil {
		return nil, err
	}
	input := &apis.StatefulsetUpdateInput{}
	if err := req.Data.Unmarshal(input); err != nil {
		return nil, err
	}
	newObj := deploy.DeepCopy()
	if input.Replicas != nil {
		newObj.Spec.Replicas = input.Replicas
	}
	template := &newObj.Spec.Template
	if err := app.UpdatePodTemplate(template, input.PodUpdateInput); err != nil {
		return nil, err
	}
	newObj.Spec.Template = *template
	cli := req.GetK8sClient()
	return cli.AppsV1().StatefulSets(newObj.GetNamespace()).Update(newObj)
}
