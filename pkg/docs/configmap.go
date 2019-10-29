package docs

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route GET /configmaps/{name} configMapShowInput
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
