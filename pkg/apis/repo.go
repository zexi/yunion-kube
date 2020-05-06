package apis

import "yunion.io/x/onecloud/pkg/apis"

const (
	RepoTypeOneCloud  = "onecloud"
	RepoTypeCommunity = "community"
)

type RepoCreateInput struct {
	apis.SharableVirtualResourceCreateInput

	// Repo URL
	// required: true
	// example: http://mirror.azure.cn/kubernetes/charts
	Url string `json:"url"`

	// Repo type
	// default: community
	Type string `json:"type"`
}
