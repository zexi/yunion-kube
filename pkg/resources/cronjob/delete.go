package cronjob

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SCronJobManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sClient()
	namespace := req.GetDefaultNamespace()
	_, err := cli.BatchV1beta1().CronJobs(namespace).Get(id, metaV1.GetOptions{})
	if err != nil {
		return err
	}
	return app.DeleteResource(req, man.KeywordPlural(), namespace, id)
}
