package docs

import (
	extensions "k8s.io/api/extensions/v1beta1"
	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route POST /ingresses ingress ingressCreateInput
// Create ingress
// responses:
// 200: ingressCreateOutput

// swagger:parameters ingressCreateInput
type ingressCreateInput struct {
	// in:body
	Body struct {
		api.K8sNamespaceResourceCreateInput
		extensions.IngressSpec
	}
}

// swagger:response ingressCreateOutput
type ingressCreateOutput struct {
	// in:body
	Body struct {
		Output extensions.Ingress
	} `json:"ingress"`
}
