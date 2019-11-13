package deployment

import (
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SDeploymentManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sManager()
	namespace := req.GetDefaultNamespace()
	indexer := cli.GetIndexer()
	deployment, err := indexer.DeploymentLister().Deployments(namespace).Get(id)
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, namespace, deployment.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.KeywordPlural(), namespace, id)
}
