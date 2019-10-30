package docs

type k8sObjectName struct {
	// The Name of kubernetes object
	// in: path
	// required: true
	Name string `json:"name"`
}
