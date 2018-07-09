package server

import (
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	"yunion.io/yunioncloud/pkg/appsrv"
	"yunion.io/yunioncloud/pkg/cloudcommon"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/clusterrouter/proxy"
	"yunion.io/yunion-kube/pkg/dialer"
	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/types/config"
)

func initCloudApp() *appsrv.Application {
	app := cloudcommon.InitApp(&options.Options.Options)
	InitHandlers(app)

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!!")
	})

	return app
}

func Start(httpsAddr string, scaledCtx *config.ScaledContext) error {
	log.Infof("Start listen on https addr: %q", httpsAddr)

	opt := options.Options

	tlsCertFile := opt.TlsCertFile
	tlsPrivateKey := opt.TlsPrivateKeyFile
	if tlsCertFile == "" || tlsPrivateKey == "" {
		return fmt.Errorf("Please specify --tls-cert-file and --tls-private-key-file")
	}

	// must before InitDB?
	app := initCloudApp()
	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()
	if !db.CheckSync(options.Options.AutoSyncTable) {
		log.Fatalf("Fail sync db")
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
	root.Handle("/connect/register", connectHandler)
	root.PathPrefix("/api/").Handler(app)

	serveHTTPS := func() error {
		return http.ListenAndServeTLS(httpsAddr, tlsCertFile, tlsPrivateKey, root)
	}
	return serveHTTPS()
}

func connectHandlers(scaledCtx *config.ScaledContext) (http.Handler, http.Handler) {
	if f, ok := scaledCtx.Dialer.(*dialer.Factory); ok {
		return f.Tunnelserver, nil
	}
	return http.NotFoundHandler(), http.NotFoundHandler()
}
