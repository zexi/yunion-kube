package k8s

import (
	"strings"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func TrimKindPlural(plural string) string {
	switch plural {
	case "ingresses":
		return api.ResourceKindIngress
	default:
		return strings.TrimRight(plural, "s")
	}
}
