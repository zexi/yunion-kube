package apis

import "yunion.io/x/onecloud/pkg/apis"

type RepoType string

const (
	RepoTypeInternal  RepoType = "internal"
	RepoTypeExternal RepoType = "external"
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

type RepoListInput struct {
	apis.SharableVirtualResourceListInput

	Type string `json:"type"`
}
