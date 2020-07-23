package models

import (
	"k8s.io/api/core/v1"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

type IEventManager interface {
	GetWarningEventsByPods(cluster model.ICluster, pods []*v1.Pod) ([]*api.Event, error)
}

var (
	eventManager IEventManager
)

func InitEventManager(em IEventManager) {
	eventManager = em
}

func GetEventManager() IEventManager {
	return eventManager
}
