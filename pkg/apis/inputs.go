package apis

type ListInputK8SBase struct {
	Limit        int64  `json:"limit"`
	Offset       int64  `json:"offset"`
	PagingMarker string `json:"paging_marker"`

	// TODO: support Label selectors

	// Name of the field to be ordered by
	OrderBy []string `json:"order_by"`
	// List order, choices 'desc|asc'
	Order  string   `json:"order"`
	Filter []string `json:"filter"`
}

type ListInputK8SClusterBase struct {
	ListInputK8SBase

	Name string `json:"name"`
}

type ListInputK8SNamespaceBase struct {
	ListInputK8SClusterBase

	Namespace string `json:"namespace"`
}
