package docs

import (
	apps "k8s.io/api/apps/v1beta2"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route PUT /deployments deploymentUpdateInput
// Update deployment spec
// responses:
// 200: deploymentUpdateOutput

// swagger:parameters deploymentUpdateInput
type deploymentUpdateInput struct {
	// in:body
	Body api.DeploymentUpdateInput
}

// swagger:response deploymentUpdateOutput
type deploymentUpdateOutput struct {
	// in:body
	Body struct {
		Output apps.Deployment `json:"deployment"`
	}
}
