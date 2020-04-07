package apis

type ListInputK8SBase struct {
	Limit        int64  `json:"limit"`
	Offset       int64  `json:"offset"`
	PagingMarker string `json:"paging_marker"`

	// Label selectors
}

type ListInputK8SClusterBase struct {
	ListInputK8SBase

	Name string `json:"name"`
}

type ListInputK8SNamespaceBase struct {
	ListInputK8SClusterBase

	Namespace string `json:"namespace"`
}
