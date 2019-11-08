package apis

import (
	//"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
)

type ReleaseCreateInput struct {
	Repo        string            `json:"repo"`
	ChartName   string            `json:"chart_name"`
	Namespace   string            `json:"namespace"`
	ReleaseName string            `json:"release_name"`
	Version     string            `json:"version"`
	Values      string            `json:"values"`
	Sets        map[string]string `json:"sets"`
}

type ReleaseUpdateInput struct {
	ReleaseCreateInput
	RecreatePods bool `json:"recreate_pods"`
	// force resource updates through a replacement strategy
	Force bool `json:"force"`
	// when upgrading, reset the values to the ones built into the chart
	ResetValues bool `json:"reset_values"`
	// when upgrading, reuse the last release's values and merge in any overrides, if reset_values is specified, this is ignored
	ReUseValues bool `json:"reuse_values"`
}

type ReleaseListQuery struct {
	Filter       string `json:"filter"`
	All          bool   `json:"all"`
	AllNamespace bool   `json:"all_namespace"`
	Namespace    string `json:"namespace"`
	Admin        bool   `json:"admin"`
	Deployed     bool   `json:"deployed"`
	Deleted      bool   `json:"deleted"`
	Deleting     bool   `json:"deleting"`
	Failed       bool   `json:"failed"`
	Superseded   bool   `json:"superseded"`
	Pending      bool   `json:"pending"`
}

type Release struct {
	*release.Release
	*ClusterMeta
}

type ReleaseDetail struct {
	Release
	Resources    map[string][]interface{} `json:"resources"`
	ConfigValues map[string]interface{}      `json:"config_values"`
}

type ReleaseHistoryInfo struct {
	Revision    int       `json:"revision"`
	Updated     time.Time `json:"updated"`
	Status      string    `json:"status"`
	Chart       string    `json:"chart"`
	Description string    `json:"description"`
}

type ReleaseRollbackInput struct {
	Revision    int    `json:"revision"`
	Description string `json:"description"`
	// will (if true) recreate pods after a rollback.
	Recreate bool `json:"recreate"`
	// will (if true) force resource upgrade through uninstall/recreate if needed
	Force bool `json:"force"`
}
