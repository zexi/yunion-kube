package helm

import (
	"yunion.io/x/yunion-kube/pkg/types/helm"
)

var OfficialRepos = []helm.Repo{
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
