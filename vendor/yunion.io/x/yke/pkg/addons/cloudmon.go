package addons

import "yunion.io/x/yke/pkg/templates"

func GetYunionCloudMoniotrManifest(config interface{}) (string, error) {
	return templates.CompileTemplateFromMap(templates.YunionCloudMonitorTemplate, config)
}