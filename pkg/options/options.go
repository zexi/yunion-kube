package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon"
)

var (
	Options KubeServerOptions
)

type KubeServerOptions struct {
	cloudcommon.DBOptions
	cloudcommon.CommonOptions

	TlsCertFile       string `help:"File containing the default x509 cert file"`
	TlsPrivateKeyFile string `help:"Tls private key"`
	HttpsPort         int    `help:"The https port that the service runs on" default:"8443"`

	HelmDataDir         string `help:"Helm data directory" default:"/opt/cloud/workspace/helm"`
	YunionChartRepo     string `help:"Yunion helm charts repo" default:"https://charts.yunion.cn"`
	RepoRefreshDuration int    `help:"Helm repo auto refresh duration, default 5 mins" default:"5"`

	LxcfsRequireAnnotation bool `help:"Only mount lxcfs volume when pod set 'initializer.kubernetes.io/lxcfs' annotation" default:"false"`

	EnableDefaultLimitRange bool `help:"Enable default namespace limit range" default:"false"`

	DockerdBip string `help:"Global nodes docker daemon bridge CIDR" default:"172.17.0.1/16"`
}
