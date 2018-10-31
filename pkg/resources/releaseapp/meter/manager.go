package meter

import (
	"yunion.io/x/yunion-kube/pkg/resources/releaseapp"
)

var MeterAppManager *SMeterAppManager

type SMeterAppManager struct {
	*releaseapp.SReleaseAppManager
}

func init() {
	MeterAppManager = &SMeterAppManager{}

	MeterAppManager.SReleaseAppManager = releaseapp.NewReleaseAppManager(MeterAppManager, "meter", "meters")
}

func (man *SMeterAppManager) GetConfigSets() releaseapp.ConfigSets {
	globalSets := releaseapp.GetYunionGlobalConfigSets()
	return globalSets
}
