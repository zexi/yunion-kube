package pod

import (
	"yunion.io/yunion-kube/pkg/resources"
)

var PodManager *SPodManager

type SPodManager struct {
	*resources.SResourceBaseManager
}

func init() {
	PodManager = &SPodManager{
		SResourceBaseManager: resources.NewResourceBaseManager("pod", "pods"),
	}
}
