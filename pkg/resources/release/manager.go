package release

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var ReleaseManager *SReleaseManager

type SReleaseManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	ReleaseManager = &SReleaseManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("release", "releases"),
	}
}
