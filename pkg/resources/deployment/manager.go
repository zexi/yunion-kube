package deployment

import (
	apps "k8s.io/api/apps/v1beta2"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources"
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var (
	DeploymentManager *SDeploymentManager
)

type SDeploymentManager struct {
	*resources.SNamespaceResourceManager
}

func (m *SDeploymentManager) IsRawResource() bool {
	return false
}

func init() {
	DeploymentManager = &SDeploymentManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("deployment", "deployments"),
	}
}

func (m *SDeploymentManager) get(req *common.Request, id string) (*apps.Deployment, error) {
	cli := req.GetK8sManager()
	namespace := req.GetDefaultNamespace()
	indexer := cli.GetIndexer()
	return indexer.DeploymentLister().Deployments(namespace).Get(id)
}

func (m *SDeploymentManager) AllowUpdateItem(req *common.Request, id string) bool {
	return m.SNamespaceResourceManager.AllowUpdateItem(req, id)
}

func (m *SDeploymentManager) Update(req *common.Request, id string) (interface{}, error) {
	deploy, err := m.get(req, id)
	if err != nil {
		return nil, err
	}
	input := &apis.DeploymentUpdateInput{}
	if err := req.Data.Unmarshal(input); err != nil {
		return nil, err
	}
	newDeploy := deploy.DeepCopy()
	if input.Replicas != nil {
		newDeploy.Spec.Replicas = input.Replicas
	}
	template := &newDeploy.Spec.Template
	if err := app.UpdatePodTemplate(template, input.PodUpdateInput); err != nil {
		return nil, err
	}
	newDeploy.Spec.Template = *template
	cli := req.GetK8sClient()
	return cli.AppsV1beta2().Deployments(newDeploy.GetNamespace()).Update(newDeploy)
}
