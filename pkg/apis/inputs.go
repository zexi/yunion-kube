package apis

type ListInputK8SBase struct {
	Limit        int64  `json:"limit"`
	Offset       int64  `json:"offset"`
	PagingMarker string `json:"paging_marker"`
	Namespace    string `json:"namespace"`

	// Label selectors
}
