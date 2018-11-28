package app

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/service"
)

func DeleteServices(cli kubernetes.Interface, namespace string, labelSelector *metav1.LabelSelector) error {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return err
	}
	options := metav1.ListOptions{LabelSelector: selector.String()}
	channels := &common.ResourceChannels{
		ServiceList: common.GetServiceListChannelWithOptions(cli, common.NewSameNamespaceQuery(namespace), options),
	}
	svcList, err := service.GetServiceListFromChannels(channels, dataselect.DefaultDataSelect())
	if err != nil {
		return err
	}
	for _, svc := range svcList.Services {
		err = cli.CoreV1().Services(namespace).Delete(svc.Name, &metav1.DeleteOptions{})
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
