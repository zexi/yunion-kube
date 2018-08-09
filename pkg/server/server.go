package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/yunionio/log"
	"github.com/yunionio/pkg/appsrv"

	"yunion.io/yunion-kube/pkg/clusterrouter"
	"yunion.io/yunion-kube/pkg/controllers"
	"yunion.io/yunion-kube/pkg/dialer"
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/types/config"
	"yunion.io/yunion-kube/pkg/ykenodeconfigserver"
)

func Start(httpsAddr string, scaledCtx *config.ScaledContext, app *appsrv.Application) error {
	log.Infof("Start listen on https addr: %q", httpsAddr)

	opt := options.Options

	tlsCertFile := opt.TlsCertFile
	tlsPrivateKey := opt.TlsPrivateKeyFile
	if tlsCertFile == "" || tlsPrivateKey == "" {
		return fmt.Errorf("Please specify --tls-cert-file and --tls-private-key-file")
	}

	root := mux.NewRouter()
	root.UseEncodedPath()

	httpRoot := mux.NewRouter()
	httpRoot.UseEncodedPath()

	proxy := clusterrouter.New()

	connectHandler, connectConfigHandler := connectHandlers(scaledCtx)

	root.PathPrefix(models.K8S_PROXY_URL_PREFIX).Handler(proxy)
	root.PathPrefix(models.K8S_AUTH_WEBHOOK_PREFIX).Handler(controllers.NewAuthHandlerFactory())
	root.Handle("/connect", connectHandler)
	root.Handle("/connect/register", connectHandler)
	root.Handle("/connect/config", connectConfigHandler)
	root.PathPrefix("/api/").Handler(app)

	serveHTTPS := func() error {
		return http.ListenAndServeTLS(httpsAddr, tlsCertFile, tlsPrivateKey, root)
	}
	return serveHTTPS()
}

func connectHandlers(scaledCtx *config.ScaledContext) (http.Handler, http.Handler) {
	if f, ok := scaledCtx.Dialer.(*dialer.Factory); ok {
		return f.Tunnelserver, ykenodeconfigserver.Handler(f.TunnelAuthorizer, scaledCtx)
	}
	return http.NotFoundHandler(), http.NotFoundHandler()
}
