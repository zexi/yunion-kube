package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/clusterrouter/proxy"
	"yunion.io/yunion-kube/pkg/dialer"
	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/types/config"
)

func Start(httpAddr, httpsAddr string, scaledCtx *config.ScaledContext) error {
	log.Infof("Start listen on https addr: %q, http addr: %q", httpsAddr, httpAddr)

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

	sp, err := proxy.New(&scaledCtx.RESTConfig, nil)
	if err != nil {
		return err
	}

	connectHandler, _ := connectHandlers(scaledCtx)

	root.PathPrefix("/k8s/clusters/").Handler(sp)
	root.Handle("/connect", connectHandler)
	//httpRoot.Handle("/connect", connectHandler)

	serveHTTPS := func() error {
		return http.ListenAndServeTLS(httpsAddr, tlsCertFile, tlsPrivateKey, root)
	}
	serveHTTP := func() {
		err := http.ListenAndServe(httpAddr, httpRoot)
		if err != nil {
			log.Fatalf("http: %v", err)
		}
	}
	go serveHTTP()
	return serveHTTPS()
}

func connectHandlers(scaledCtx *config.ScaledContext) (http.Handler, http.Handler) {
	if f, ok := scaledCtx.Dialer.(*dialer.Factory); ok {
		return f.Tunnelserver, nil
	}
	return http.NotFoundHandler(), http.NotFoundHandler()
}
