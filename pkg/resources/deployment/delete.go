package deployment

import (
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SDeploymentManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sManager()
	namespace := req.GetDefaultNamespace()
	deployment, err := cli.GetIndexer().DeploymentLister().Deployments(namespace).Get(id)
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, req.GetCluster(), namespace, deployment.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.Keyword(), namespace, id)
}
