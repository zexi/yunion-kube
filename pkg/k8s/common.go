package k8s

import (
	"encoding/json"
	"net/http"

	"yunion.io/x/log"
)

func SendJSON(w http.ResponseWriter, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(obj)
	if err != nil {
		log.Errorf("Send obj %#v to http response error: %v", obj, obj)
	}
}
