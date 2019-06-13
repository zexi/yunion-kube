package addons

import (
	"bytes"
	"text/template"

	"github.com/pkg/errors"
)

func CompileTemplateFromMap(tmplt string, configMap interface{}) (string, error) {
	out := new(bytes.Buffer)
	t := template.Must(template.New("compiled_template").Parse(tmplt))
	if err := t.Execute(out, configMap); err != nil {
		return "", err
	}
	return out.String(), nil
}

const YunionVMManifestTemplate = `
#### CNIPlugin config ####
---
{{.CNIPlugin}}
---
#### CSIPlugin config ####
---
{{.CSIPlugin}}
---
#### MetricsPlugin config ####
---
{{.MetricsPlugin}}
---
#### HelmPlugin config ####
---
{{.HelmPlugin}}
---
#### CloudProviderPlugin config ####
---
{{.CloudProviderPlugin}}
---
#### IngressControllerPlugin config ####
---
{{.IngressControllerPlugin}}
---
`

type yunionConfig struct {
	CNIPlugin               string
	CSIPlugin               string
	MetricsPlugin           string
	HelmPlugin              string
	CloudProviderPlugin     string
	IngressControllerPlugin string
}

func (c yunionConfig) GenerateYAML() (string, error) {
	return CompileTemplateFromMap(YunionVMManifestTemplate, c)
}

type YunionCommonPluginsConfig struct {
	*MetricsPluginConfig
	*HelmPluginConfig
	*CloudProviderYunionConfig
	*CSIYunionConfig
	*IngressControllerYunionConfig
}

func (config *YunionCommonPluginsConfig) GetAllConfig() (*yunionConfig, error) {
	allConfig := new(yunionConfig)
	if config.MetricsPluginConfig != nil {
		ret, err := config.MetricsPluginConfig.GenerateYAML()
		if err != nil {
			return nil, errors.Wrap(err, "Generate metrics plugin")
		}
		allConfig.MetricsPlugin = ret
	}
	if config.HelmPluginConfig != nil {
		ret, err := config.HelmPluginConfig.GenerateYAML()
		if err != nil {
			return nil, errors.Wrap(err, "Generate helm plugin")
		}
		allConfig.HelmPlugin = ret
	}
	if config.CloudProviderYunionConfig != nil {
		ret, err := config.CloudProviderYunionConfig.GenerateYAML()
		if err != nil {
			return nil, errors.Wrap(err, "Generate cloud provider plugin")
		}
		allConfig.CloudProviderPlugin = ret
	}
	if config.CSIYunionConfig != nil {
		ret, err := config.CSIYunionConfig.GenerateYAML()
		if err != nil {
			return nil, errors.Wrap(err, "Generate csi plugin")
		}
		allConfig.CSIPlugin = ret
	}
	if config.IngressControllerYunionConfig != nil {
		ret, err := config.IngressControllerYunionConfig.GenerateYAML()
		if err != nil {
			return nil, errors.Wrap(err, "Generate csi plugin")
		}
		allConfig.IngressControllerPlugin = ret
	}
	return allConfig, nil
}

type YunionVMPluginsConfig struct {
	*YunionCommonPluginsConfig
	*CNICalicoConfig
}

func (c *YunionVMPluginsConfig) GenerateYAML() (string, error) {
	allConfig, err := c.YunionCommonPluginsConfig.GetAllConfig()
	if err != nil {
		return "", errors.Wrap(err, "get allConfig")
	}
	if c.CNICalicoConfig != nil {
		ret, err := c.CNICalicoConfig.GenerateYAML()
		if err != nil {
			return "", errors.Wrap(err, "Generate calico cni")
		}
		allConfig.CNIPlugin = ret
	}
	return allConfig.GenerateYAML()
}

type YunionHostPluginsConfig struct {
	*YunionCommonPluginsConfig
	*CNIYunionConfig
}

func (c *YunionHostPluginsConfig) GenerateYAML() (string, error) {
	allConfig, err := c.YunionCommonPluginsConfig.GetAllConfig()
	if err != nil {
		return "", errors.Wrap(err, "get allConfig")
	}
	if c.CNIYunionConfig != nil {
		ret, err := c.CNIYunionConfig.GenerateYAML()
		if err != nil {
			return "", errors.Wrap(err, "Generate yunion cni")
		}
		allConfig.CNIPlugin = ret
	}
	return allConfig.GenerateYAML()
}
