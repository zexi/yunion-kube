package statefulset

import (
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SStatefuleSetManager) Delete(req *common.Request, id string) error {
	cli := req.GetK8sManager()
	namespace := req.GetDefaultNamespace()
	ss, err := cli.GetIndexer().StatefulSetLister().StatefulSets(namespace).Get(id)
	if err != nil {
		return err
	}
	err = app.DeleteServices(cli, req.GetCluster(), namespace, ss.Spec.Selector)
	if err != nil {
		return err
	}

	return app.DeleteResource(req, man.Keyword(), namespace, id)
}
