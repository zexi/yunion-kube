package utils

import (
	"gopkg.in/yaml.v2"

	"yunion.io/yke/pkg/types"
)

func ConvertToYkeConfig(config string) (types.KubernetesEngineConfig, error) {
	var conf types.KubernetesEngineConfig
	if err := yaml.Unmarshal([]byte(config), &conf); err != nil {
		return conf, err
	}
	return conf, nil
}

func ConvertYkeConfigToStr(conf types.KubernetesEngineConfig) (string, error) {
	bytes, err := yaml.Marshal(conf)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
