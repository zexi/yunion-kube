package apis

import (
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type RegistrySecretCreateInput struct {
	K8sNamespaceResourceCreateInput

	// required: true
	User string `json:"user"`
	// required: true
	Password string `json:"password"`
	// required: true
	Server string `json:"server"`
	Email  string `json:"email"`
}

// Secret is a single secret returned to the frontend.
type Secret struct {
	api.ObjectMeta
	api.TypeMeta
	Type v1.SecretType `json:"type"`
}

// SecretDetail API resource provides mechanisms to inject containers with configuration data while keeping
// containers agnostic of Kubernetes
type SecretDetail struct {
	Secret

	// Data contains the secret data.  Each key must be a valid DNS_SUBDOMAIN
	// or leading dot followed by valid DNS_SUBDOMAIN.
	// The serialized form of the secret data is a base64 encoded string,
	// representing the arbitrary (possibly non-string) data value here.
	Data map[string][]byte `json:"data"`
}
