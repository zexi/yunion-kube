package controllers

import (
	"net/http"
	"strings"

	"yunion.io/x/log"
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
	log.Errorf("-----+++ get ctrl")
	ctrl, err := Manager.GetController(clusterId)
	if err != nil {
		return nil, err
	}
	log.Errorf("----- get ctrl")
	return ctrl.GetKeystoneAuthenticator(), nil
}

func (f *authFactory) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Auth request url: %s", r.URL)
	clusterId := GetClusterId(r)
	if clusterId == "" {
		httperrors.NotAcceptableError(w, "Cluster id not provide")
		return
	}
	cluster, err := models.ClusterManager.FetchClusterByIdOrName("", clusterId)
	if err != nil {
		httperrors.NotFoundError(w, err.Error())
		return
	}
	log.Errorf("=====get keystone authenticator: %s", r.URL)
	kauth, err := f.getKeystoneAuthenticator(cluster.Id)
	if err != nil {
		log.Errorf("xxxxxx get authticator error: %v", err)
		httperrors.NotFoundError(w, err.Error())
		return
	}
	h := &auth.WebhookHandler{kauth}
	h.ServeHTTP(w, r)
}
