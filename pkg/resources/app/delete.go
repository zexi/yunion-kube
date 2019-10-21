package app

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/yunion-kube/pkg/client"
	capi "yunion.io/x/yunion-kube/pkg/client/api"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func GetRelatedServices(indexer *client.CacheFactory, namespace string, labelSelector *metav1.LabelSelector) ([]*v1.Service, error) {
	svcs, err := indexer.ServiceLister().Services(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}
	ret := make([]*v1.Service, 0)
	for _, svc := range svcs {
		if reflect.DeepEqual(svc.Spec.Selector, labelSelector.MatchLabels) {
			ret = append(ret, svc)
		}
	}
	return ret, nil
}

func DeleteServices(cli *client.ClusterManager, namespace string, labelSelector *metav1.LabelSelector) error {
	svcList, err := GetRelatedServices(cli.GetIndexer(), namespace, labelSelector)
	if err != nil {
		return err
	}
	for _, svc := range svcList {
		err = cli.KubeClient.Delete(capi.ResourceNameService, namespace, svc.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func DeleteResource(req *common.Request, controllerType, namespace, name string) error {
	verber, err := req.GetVerberClient()
	if err != nil {
		return err
	}
	return verber.Delete(controllerType, true, namespace, name)
}
