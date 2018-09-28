package models

import (
	"fmt"

	"yunion.io/x/pkg/util/sets"
)

const (
	K8S_V1_10_5         = "v1.10.5"
	K8S_V1_11_2         = "v1.11.2"
	DEFAULT_K8S_VERSION = K8S_V1_10_5

	YKE_K8S_V1_10_5 = "v1.10.5-rancher1-2"
	YKE_K8S_V1_11_2 = "v1.11.2-rancher1-1"
)

var (
	SupportVersions = sets.NewString(K8S_V1_10_5, K8S_V1_11_2)

	K8sYKEVersionMap = ykeVersionMap{
		K8S_V1_10_5: YKE_K8S_V1_10_5,
		K8S_V1_11_2: YKE_K8S_V1_11_2,
	}

	YKEK8sVersionMap = map[string]string{
		YKE_K8S_V1_10_5: K8S_V1_10_5,
		YKE_K8S_V1_11_2: K8S_V1_11_2,
	}
)

type ykeVersionMap map[string]string

func (m ykeVersionMap) GetYKEVersion(v string) (string, error) {
	if !SupportVersions.Has(v) {
		return "", fmt.Errorf("Not support version: %q", v)
	}
	return m[v], nil
}
