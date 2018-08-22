package k8s

import (
	"encoding/json"
	"net/http"
	"strings"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func TrimKindPlural(plural string) string {
	switch plural {
	case "ingresses":
		return api.ResourceKindIngress
	case "k8s_services":
		return api.ResourceKindService
	default:
		return strings.TrimRight(plural, "s")
	}
}

func SendJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(obj)
	if err != nil {
		log.Errorf("Send obj %#v to http response error: %v", obj, obj)
	}
}
