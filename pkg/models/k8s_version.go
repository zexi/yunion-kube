package models

import (
	"fmt"

	"yunion.io/x/pkg/util/sets"
)

const (
	K8S_V1_10_5         = "v1.10.5"
	K8S_V1_11_3         = "v1.11.3"
	K8S_V1_12_0         = "v1.12.0"
	DEFAULT_K8S_VERSION = K8S_V1_12_0

	YKE_K8S_V1_10_5 = "v1.10.5-rancher1-2"
	YKE_K8S_V1_11_3 = "v1.11.3-rancher1-1"
	YKE_K8S_V1_12_0 = "v1.12.0-rancher1-1"
)

var (
	SupportVersions = sets.NewString(K8S_V1_10_5, K8S_V1_11_3, K8S_V1_12_0)

	K8sYKEVersionMap = ykeVersionMap{
		K8S_V1_10_5: YKE_K8S_V1_10_5,
		K8S_V1_11_3: YKE_K8S_V1_11_3,
		K8S_V1_12_0: YKE_K8S_V1_12_0,
	}

	YKEK8sVersionMap = map[string]string{
		YKE_K8S_V1_10_5: K8S_V1_10_5,
		YKE_K8S_V1_11_3: K8S_V1_11_3,
		YKE_K8S_V1_12_0: K8S_V1_12_0,
	}
)

type ykeVersionMap map[string]string

func (m ykeVersionMap) GetYKEVersion(v string) (string, error) {
	if !SupportVersions.Has(v) {
		return "", fmt.Errorf("Not support version: %q", v)
	}
	return m[v], nil
}
