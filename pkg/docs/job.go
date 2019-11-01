package docs

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route POST /jobs job jobCreateInput
// Create job
// responses:
// 200: jobCreateOutput

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

// swagger:route POST /cronjobs cronjob cronjobCreateInput
// Create CronJob
// responses:
// 200: cronjobCreateOutput

// swagger:parameters cronjobCreateInput
type cronjobCreateInput struct {
	// in:body
	Body api.CronJobCreateInput
}

// swagger:response cronjobCreateOutput
type cronjobCreateOutput struct {
	// in:body
	Body struct {
		Output v1beta1.CronJob `json:"cronjob"`
	}
}
