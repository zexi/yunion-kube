package pod

import (
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

func GetEventsForPod(client client.Interface, dsQuery *dataselect.DataSelectQuery, namespace,
	podName string) (*common.EventList, error) {
	eventList := common.EventList{
		ListMeta: dataselect.NewListMeta(),
		Events:   make([]common.Event, 0),
	}
	podEvents, err := event.GetPodEvents(client, namespace, podName)
	if err != nil {
		return &eventList, err
	}
	eventList, err = event.CreateEventList(podEvents, dsQuery)
	if err != nil {
		return &eventList, err
	}

	log.Infof("Found %d events related to %s pod in %s namespace", len(eventList.Events), podName, namespace)
	return &eventList, nil
}
