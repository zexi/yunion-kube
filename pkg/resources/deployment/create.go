package deployment

import (
	apps "k8s.io/api/apps/v1beta2"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SDeploymentManager) ValidateCreateData(req *common.Request) error {
	return app.ValidateCreateData(req, man)
}

func (man *SDeploymentManager) Create(req *common.Request) (interface{}, error) {
	return createDeploymentApp(req)
}

func createDeploymentApp(req *common.Request) (*apps.Deployment, error) {
	objMeta, selector, err := common.GetK8sObjectCreateMetaWithLabel(req)
	if err != nil {
		return nil, err
	}
	input := &api.DeploymentCreateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	input.Template.ObjectMeta = *objMeta
	input.Selector = selector

	deployment := &apps.Deployment{
		ObjectMeta: *objMeta,
		Spec:       input.DeploymentSpec,
	}

	if _, err := common.CreateServiceIfNotExist(req, objMeta, input.Service); err != nil {
		return nil, err
	}

	return req.GetK8sClient().AppsV1beta2().Deployments(deployment.Namespace).Create(deployment)
}
