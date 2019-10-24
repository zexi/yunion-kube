package docs

import api "yunion.io/x/yunion-kube/pkg/apis"

// swagger:route POST /kubeclusters clusterCreateInput
// Create kubernetes cluster
// responses:
//   200: clusterCreateOutput

// swagger:parameters clusterCreateInput
type clusterCreateInput struct {
	// in:body
	Body api.ClusterCreateInput
}

// swagger:response clusterCreateOutput
type clusterCreateOutput struct {
	// in:body
	Body struct {
		Output api.ClusterCreateOutput `json:"kubecluster"`
	}
}
