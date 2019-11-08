package apis

import (
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/repo"
)

type ChartResult struct {
	*repo.ChartVersion `json:"chart"`
	Repo               string `json:"repo"`
}

type ChartListInput struct {
	Name       string `json:"name"`
	Repo       string `json:"repo"`
	AllVersion bool   `json:"all_version"`
	Keyword    string `json:"keyword"`
}

type SpotguideOptions struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Default bool   `json:"default"`
	Info    string `json:"info"`
	Key     string `json:"key"`
}

type SpotguideFile struct {
	Options []SpotguideOptions `json:"options"`
}

type ChartDetail struct {
	*chart.Chart `json:"chart"`
	Name         string             `json:"name"`
	Repo         string             `json:"repo"`
	Readme       string             `json:"readme"`
	Options      []SpotguideOptions `json:"options"`
}