package vm

import (
	"github.com/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/drivers/clusters/addons"
)

const YunionVMManifestTemplate = `
---
{{.CNIPlugin}}
---
{{.CSIPlugin}}
---
{{.MetricsPlugin}}
---
{{.HelmPlugin}}
---
{{.CloudProviderPlugin}}
---
`

type yunionVMConfig struct {
	CNIPlugin           string
	CSIPlugin           string
	MetricsPlugin       string
	HelmPlugin          string
	CloudProviderPlugin string
}

func (c yunionVMConfig) GenerateYAML() (string, error) {
	return addons.CompileTemplateFromMap(YunionVMManifestTemplate, c)
}

type YunionVMPluginsConfig struct {
	*addons.CNICalicoConfig
	*addons.MetricsPluginConfig
	*addons.HelmPluginConfig
	*addons.CloudProviderYunionConfig
}

func GetYunionManifest(config *YunionVMPluginsConfig) (string, error) {
	if config == nil {
		return "", nil
	}
	allConfig := new(yunionVMConfig)
	if config.CNICalicoConfig != nil {
		ret, err := config.CNICalicoConfig.GenerateYAML()
		if err != nil {
			return "", errors.Wrap(err, "Generate calico cni")
		}
		allConfig.CNIPlugin = ret
	}
	if config.MetricsPluginConfig != nil {
		ret, err := config.MetricsPluginConfig.GenerateYAML()
		if err != nil {
			return "", errors.Wrap(err, "Generate metrics plugin")
		}
		allConfig.MetricsPlugin = ret
	}
	if config.HelmPluginConfig != nil {
		ret, err := config.HelmPluginConfig.GenerateYAML()
		if err != nil {
			return "", errors.Wrap(err, "Generate helm plugin")
		}
		allConfig.HelmPlugin = ret
	}
	if config.CloudProviderYunionConfig != nil {
		ret, err := config.CloudProviderYunionConfig.GenerateYAML()
		if err != nil {
			return "", errors.Wrap(err, "Generate cloud provider plugin")
		}
		allConfig.CloudProviderPlugin = ret
	}
	return allConfig.GenerateYAML()
}
