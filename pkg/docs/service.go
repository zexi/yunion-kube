package docs

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route POST /services service serviceCreateInput
// Create kubernetes service
// responses:
//   200: serviceCreateOutput

// swagger:parameters serviceCreateInput
type serviceCreateInput struct {
	// in:body
	Body api.ServiceCreateInput
}

// swagger:response serviceCreateOutput
type serviceCreateOutput struct {
	// in:body
	Body struct {
		Output api.ServiceCreateInput `json:"service"`
	}
}
