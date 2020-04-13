package k8smodels

import "yunion.io/x/yunion-kube/pkg/k8s/common/model"

var (
	ReleaseManager *SReleaseManager
)

func init() {
	ReleaseManager = &SReleaseManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SRelease), "release", "releases"),
	}
}

type SReleaseManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SRelease struct {
	model.SK8SNamespaceResourceBase
}
