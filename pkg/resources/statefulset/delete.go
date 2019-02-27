package statefulset

import (
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SStatefuleSetManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sClient()
	namespace := req.GetDefaultNamespace()
	ss, err := cli.AppsV1beta2().StatefulSets(namespace).Get(id, metaV1.GetOptions{})
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, req.GetCluster(), namespace, ss.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.Keyword(), namespace, id)
}
