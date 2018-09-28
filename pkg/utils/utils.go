package utils

import (
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"yunion.io/x/yke/pkg/types"
)

func ConvertToYkeConfig(config string) (*types.KubernetesEngineConfig, error) {
	var conf types.KubernetesEngineConfig
	if err := yaml.Unmarshal([]byte(config), &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func ConvertYkeConfigToStr(conf types.KubernetesEngineConfig) (string, error) {
	bytes, err := yaml.Marshal(conf)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func GetK8sRestConfigFromBytes(bs []byte) (*rest.Config, error) {
	config, err := clientcmd.Load(bs)
	if err != nil {
		return nil, err
	}
	return clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
}
