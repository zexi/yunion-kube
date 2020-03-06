package docs

import (
	apps "k8s.io/api/apps/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route PUT /deployments deployment deploymentUpdateInput
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

// swagger:route POST /deployments deployment deploymentCreateInput
// Create deployment
// responses:
// 200: deploymentCreateOutput

// swagger:parameters deploymentCreateInput
type deploymentCreateInput struct {
	api.DeploymentCreateInput
}

// swagger:response deploymentCreateOutput
type deploymentCreateOutput struct {
	// in:body
	Body struct {
		Output apps.Deployment `json:"deployment"`
	}
}
