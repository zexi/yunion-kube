package releaseapp

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/release"
)

type IReleaseAppHooker interface {
	GetConfigSets() ConfigSets
}

type SReleaseAppManager struct {
	*release.SReleaseManager
	hooker IReleaseAppHooker
}

func NewReleaseAppManager(hooker IReleaseAppHooker, keyword, keywordPlural string) *SReleaseAppManager {
	return &SReleaseAppManage{
		SReleaseManager: &release.SReleaseManager{
			SNamespaceResourceManage: resources.NewNamespaceResourceManager(keyword, keywordPlural),
		},
		hooker: hooker,
	}
}

type ConfigSets map[string]string

func NewConfigSets(conf map[string]string) ConfigSets {
	return ConfigSets(conf)
}

func (s ConfigSets) ToSets() []string {
	ret := make([]string, 0)
	for k, v := range s {
		ret = append(ret, fmt.Sprintf("%s=%s", k, v))
	}
	return ret
}

func GetYunionGlobalConfigSets() ConfigSets {
	o := options.Options
	return map[string]string{
		"global.yunion.auth.url":      o.AuthURL,
		"global.yunion.auth.domain":   "Default",
		"global.yunion.auth.username": o.AdminUser,
		"global.yunion.auth.password": o.AdminPassword,
		"global.yunion.auth.project":  o.AdminProject,
		"global.yunion.auth.region":   o.Region,
	}
}
