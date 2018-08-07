package helm

import (
	"yunion.io/yunion-kube/pkg/types/helm"
)

var OfficialRepos = []Repo{
	{
		Name:   "stable",
		Url:    "https://kubernetes-charts.storage.googleapis.com",
		Source: "https://github.com/kubernetes/charts/tree/master/stable",
	},
	{
		Name:   "incubator",
		Url:    "https://kubernetes-charts-incubator.storage.googleapis.com",
		Source: "https://github.com/kubernetes/charts/tree/master/incubator",
	},
}
