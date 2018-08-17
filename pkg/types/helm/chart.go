package helm

import (
	helmchart "k8s.io/helm/pkg/proto/hapi/chart"
)

type ChartPackage struct {
	*helmchart.Chart
	Repo string `json:"repo"`
}
