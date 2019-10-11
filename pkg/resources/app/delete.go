package app

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/client"
	capi "yunion.io/x/yunion-kube/pkg/client/api"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/service"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func DeleteServices(cli *client.ClusterManager, cluster api.ICluster, namespace string, labelSelector *metav1.LabelSelector) error {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return err
	}
	channels := &common.ResourceChannels{
		ServiceList: common.GetServiceListChannelWithOptions(cli.GetIndexer(), common.NewSameNamespaceQuery(namespace), selector),
	}
	svcList, err := service.GetServiceListFromChannels(channels, dataselect.DefaultDataSelect(), cluster)
	if err != nil {
		return err
	}
	for _, svc := range svcList.Services {
		err = cli.KubeClient.Delete(capi.KindNameService, namespace, svc.Name, &metav1.DeleteOptions{})
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
