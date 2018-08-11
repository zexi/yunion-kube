package models

import (
	"fmt"

	"yunion.io/x/pkg/util/sets"

	yketypes "yunion.io/yke/pkg/types"
)

const (
	K8S_V1_8_10         = "v1.8.10"
	K8S_V1_9_5          = "v1.9.5"
	K8S_V1_10_0         = "v1.10.0"
	DEFAULT_K8S_VERSION = K8S_V1_9_5
)

var (
	SupportVersions  = sets.NewString(K8S_V1_8_10, K8S_V1_9_5, K8S_V1_10_0)
	K8sYKEVersionMap = ykeVersionMap{
		K8S_V1_8_10: yketypes.K8sV18,
		K8S_V1_9_5:  yketypes.K8sV19,
		K8S_V1_10_0: yketypes.K8sV110,
	}
)

type ykeVersionMap map[string]string

func (m ykeVersionMap) GetYKEVersion(v string) (string, error) {
	if !SupportVersions.Has(v) {
		return "", fmt.Errorf("Not support version: %q", v)
	}
	return m[v], nil
}
