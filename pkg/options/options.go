package options

import (
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

var (
	Options KubeServerOptions
)

type KubeServerOptions struct {
	common_options.DBOptions
	common_options.CommonOptions

	TlsCertFile       string `help:"File containing the default x509 cert file"`
	TlsPrivateKeyFile string `help:"Tls private key"`
	HttpsPort         int    `help:"The https port that the service runs on" default:"8443"`

	HelmDataDir         string `help:"Helm data directory" default:"/opt/cloud/workspace/helm"`
	StableChartRepo     string `help:"Yunion helm charts repo" default:"http://mirror.azure.cn/kubernetes/charts/"`
	RepoRefreshDuration int    `help:"Helm repo auto refresh duration, default 5 mins" default:"5"`

	EnableDefaultLimitRange bool `help:"Enable default namespace limit range" default:"false"`

	DockerdBip string `help:"Global nodes docker daemon bridge CIDR" default:"172.17.0.1/16"`

	GuestDefaultTemplate string `help:"Guest kubernetes default image id" default:"k8s-centos7-base.qcow2"`
}

func OnOptionsChange(oldO, newO interface{}) bool {
	oldOpts := oldO.(*KubeServerOptions)
	newOpts := newO.(*KubeServerOptions)
	changed := false
	if common_options.OnCommonOptionsChange(&oldOpts.CommonOptions, &newOpts.CommonOptions) {
		changed = true
	}
	return changed
}
