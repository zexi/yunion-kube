package controllers

import (
	"net/http"
	"strings"

	"yunion.io/x/onecloud/pkg/httperrors"

	"yunion.io/x/yunion-kube/pkg/controllers/auth"
	"yunion.io/x/yunion-kube/pkg/models"
)

type authFactory struct{}

func NewAuthHandlerFactory() http.Handler {
	return &authFactory{}
}

func GetClusterId(req *http.Request) string {
	clusterId := req.Header.Get("X-API-Cluster-Id")
	if clusterId != "" {
		return clusterId
	}

	parts := strings.Split(req.URL.Path, "/")
	if len(parts) > 3 && strings.HasPrefix(parts[2], "auth") {
		return parts[3]
	}

	return ""
}

func (f *authFactory) getKeystoneAuthenticator(clusterId string) (*auth.KeystoneAuthenticator, error) {
	ctrl, err := Manager.GetController(clusterId)
	if err != nil {
		return nil, err
	}
	return ctrl.GetKeystoneAuthenticator(), nil
}

func (f *authFactory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clusterId := GetClusterId(r)
	if clusterId == "" {
		httperrors.NotAcceptableError(w, "Cluster id not provide")
		return
	}
	cluster, err := models.ClusterManager.FetchClusterById(clusterId)
	if err != nil {
		httperrors.NotFoundError(w, err.Error())
		return
	}
	kauth, err := f.getKeystoneAuthenticator(cluster.Id)
	if err != nil {
		httperrors.NotFoundError(w, err.Error())
		return
	}
	h := &auth.WebhookHandler{kauth}
	h.ServeHTTP(w, r)
}
