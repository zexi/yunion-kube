package docs

import (
	api "yunion.io/x/yunion-kube/pkg/apis"
)

// swagger:route POST /registrysecrets registrysecret registrySecretCreateInput
// Create docker registry auth secret
// responses:
//   200: registrySecretCreateOutput

// swagger:parameters registrySecretCreateInput
type registrySecretCreateInput struct {
	// in:body
	Body api.RegistrySecretCreateInput
}

// swagger:response registrySecretCreateOutput
type registrySecretCreateOutput struct {
	// in:body
	Body struct {
		Output api.SecretDetail `json:"registrysecret"`
	}
}
