package deployment

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var (
	DeploymentManager     *SDeploymentManager
	DeployFromFileManager *SDeployFromFileManager
)

type SDeploymentManager struct {
	*resources.SNamespaceResourceManager
}

type SDeployFromFileManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	DeploymentManager = &SDeploymentManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("deployment", "deployments"),
	}

	DeployFromFileManager = &SDeployFromFileManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("deployfromfile", "deployfromfiles"),
	}
}
