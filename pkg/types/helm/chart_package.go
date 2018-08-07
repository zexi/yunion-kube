package helm

type ChartPackage struct {
	AppVersion  string       `json:"appVersion"`
	Created     string       `json:"created"`
	Deprecated  bool         `json:"deprecated"`
	Description string       `json:"description"`
	Digest      string       `json:"digest"`
	Home        string       `json:"home"`
	Icon        string       `json:"icon,omitempty"`
	Keywords    []string     `json:"keywords,omitempty"`
	Maintainers []*Maintainer `json:"maintainers"`
	Name        string       `json:"name"`
	Repo        string       `json:"repo,omitempty"`
	Sources     []string     `json:"sources"`
	Urls        []string     `json:"urls"`
	Version     string       `json:"version"`
}
