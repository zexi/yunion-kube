package docs

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route GET /configmaps/{name} configmap configMapShowInput
// Get configmap details
// responses:
//   200: configMapShowOutput

// swagger:parameters configMapShowInput
type configMapShowInput struct {
	k8sObjectName
	// in:query
	api.K8sNamespaceResourceGetInput
}

// swagger:response configMapShowOutput
type configMapShowOutput struct {
	// in:body
	Body struct {
		Output api.ConfigMapDetail `json:"configmap"`
	}
}

// swagger:route POST /configmaps configmap configMapCreateInput
// Create configmap
// responses:
//   200: configMapCreateOutput

// swagger:parameters configMapCreateInput
type configMapCreateInput struct {
	// in:body
	Body api.ConfigMapCreateInput
}

// swagger:response configMapCreateOutput
type configMapCreateOutput struct {
	// in:body
	Body struct {
		Output v1.ConfigMap `json:"configmap"`
	}
}
