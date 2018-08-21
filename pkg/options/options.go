package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

var (
	Options KubeServerOptions
)

type KubeServerOptions struct {
	cloudcommon.DBOptions

	TlsCertFile       string `help:"File containing the default x509 cert file"`
	TlsPrivateKeyFile string `help:"Tls private key"`
	HttpsPort         int    `help:"The https port that the service runs on" default:"8443"`

	HelmDataDir string `help:"Helm data directory" default:"/opt/helm"`
}
