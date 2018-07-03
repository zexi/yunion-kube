package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/clusterrouter/proxy"
	"yunion.io/yunion-kube/pkg/k8s"
	"yunion.io/yunion-kube/pkg/options"
)

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")
}

func Run() error {
	prepareEnv()
	opt := options.Options
	addr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))
	log.Infof("Start listen on %q", addr)

	root := mux.NewRouter()
	root.UseEncodedPath()

	if opt.KubeConfig == "" {
		return fmt.Errorf("kube config file must provided")
	}
	kubeConfig, err := k8s.GetConfig(opt.KubeConfig)
	if err != nil {
		return err
	}

	tlsCertFile := opt.TlsCertFile
	tlsPrivateKey := opt.TlsPrivateKeyFile
	if tlsCertFile == "" || tlsPrivateKey == "" {
		return fmt.Errorf("Please specify --tls-cert-file and --tls-private-key-file")
	}

	sp, err := proxy.New(kubeConfig, nil)
	if err != nil {
		return err
	}
	root.PathPrefix("/k8s/clusters/").Handler(sp)

	return http.ListenAndServeTLS(addr, tlsCertFile, tlsPrivateKey, root)
}
