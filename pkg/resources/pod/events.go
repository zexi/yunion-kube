package pod

import (
	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func GetEventsForPod(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace,
	podName string) (*common.EventList, error) {
	eventList := &common.EventList{
		BaseList: common.NewBaseList(cluster),
		Events:   make([]common.Event, 0),
	}
	podEvents, err := event.GetPodEvents(indexer, namespace, podName)
	if err != nil {
		return eventList, err
	}
	eventList, err = event.CreateEventList(podEvents, dsQuery, cluster)
	if err != nil {
		return eventList, err
	}

	log.Infof("Found %d events related to %s pod in %s namespace", len(eventList.Events), podName, namespace)
	return eventList, nil
}
