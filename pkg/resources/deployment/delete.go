package deployment

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SDeploymentManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sClient()
	namespace := req.GetDefaultNamespace()
	deployment, err := cli.AppsV1beta2().Deployments(namespace).Get(id, metaV1.GetOptions{})
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, req.GetCluster(), namespace, deployment.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.Keyword(), namespace, id)
}
