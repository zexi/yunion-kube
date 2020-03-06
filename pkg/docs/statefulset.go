package docs

import (
	apps "k8s.io/api/apps/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route PUT /statefulsets statefulset statefulsetUpdateInput
// Update statefulset spec
// responses:
// 200: statefulsetUpdateOutput

// swagger:parameters statefulsetUpdateInput
type statefulsetUpdateInput struct {
	// in:body
	Body api.StatefulsetUpdateInput
}

// swagger:response statefulsetUpdateOutput
type statefulsetUpdateOutput struct {
	// in:body
	Body struct {
		Output apps.Deployment `json:"statefulset"`
	}
}

// swagger:route POST /statefulsets statefulset statefulsetCreateInput
// Create statefulset
// responses:
// 200: statefulsetCreateOutput

// swagger:parameters statefulsetCreateInput
type statefulsetCreateInput struct {
}

// swagger:response statefulsetCreateOutput
type statefulsetCreateOutput struct {
	// in:body
	Body struct {
		Output apps.StatefulSet `json:"statefulset"`
	}
}
