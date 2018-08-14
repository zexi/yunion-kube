package deployment

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var DeploymentManager *SDeploymentManager

type SDeploymentManager struct {
	*resources.SResourceBaseManager
}

func init() {
	DeploymentManager = &SDeploymentManager{
		SResourceBaseManager: resources.NewResourceBaseManager("deployment", "deployments"),
	}
}
