package deployment

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var DeploymentManager *SDeploymentManager

type SDeploymentManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	DeploymentManager = &SDeploymentManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("deployment", "deployments"),
	}
}
