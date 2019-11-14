package job

import (
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SJobManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sManager()
	namespace := req.GetDefaultNamespace()
	job, err := cli.GetIndexer().JobLister().Jobs(namespace).Get(id)
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, namespace, job.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.KeywordPlural(), namespace, id)
}
