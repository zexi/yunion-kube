package docs

import (
	batch "k8s.io/api/batch/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route POST /jobs job jobCreateInput
// Create job
// responses:
// 100: jobCreateOutput

// swagger:parameters jobCreateInput
type jobCreateInput struct {
	// in:body
	Body api.JobCreateInput
}

// swagger:response jobCreateOutput
type jobCreateOutput struct {
	// in:body
	Body struct {
		Output batch.Job `json:"job"`
	}
}
