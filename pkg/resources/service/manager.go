package service

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var ServiceManager *SServiceManager

type SServiceManager struct {
	*resources.SResourceBaseManager
}

func init() {
	ServiceManager = &SServiceManager{
		SResourceBaseManager: resources.NewResourceBaseManager("service", "services"),
	}
}
