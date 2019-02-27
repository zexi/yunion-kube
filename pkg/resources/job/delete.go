package job

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SJobManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sClient()
	namespace := req.GetDefaultNamespace()
	job, err := cli.BatchV1().Jobs(namespace).Get(id, metaV1.GetOptions{})
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, req.GetCluster(), namespace, job.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.Keyword(), namespace, id)
}
