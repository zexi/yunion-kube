package deployment

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SDeploymentManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sClient()
	namespace := req.GetDefaultNamespace()
	deployment, err := GetDeploymentDetail(cli, namespace, id)
	if err != nil {
		return err
	}

	deleteOpt := &metaV1.DeleteOptions{}
	// delete services
	for _, svc := range deployment.ServiceList {
		err = cli.CoreV1().Services(namespace).Delete(svc.Name, deleteOpt)
		if err != nil {
			return err
		}
	}

	verber, err := req.GetVerberClient()
	if err != nil {
		return err
	}

	return verber.Delete(man.Keyword(), true, namespace, id)
}
