package service

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var ServiceManager *SServiceManager

type SServiceManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ServiceManager = &SServiceManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("k8s_service", "k8s_services"),
	}
}
