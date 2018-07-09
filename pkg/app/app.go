package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	"k8s.io/client-go/rest"

	"yunion.io/yunioncloud/pkg/cloudcommon"

	"yunion.io/yunion-kube/pkg/dialer"
	"yunion.io/yunion-kube/pkg/k8s"
	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/server"
	"yunion.io/yunion-kube/pkg/types/config"
)

func buildScaledContext(ctx context.Context, kubeConfig rest.Config) (*config.ScaledContext, error) {
	scaledCtx, err := config.NewScaledContext(kubeConfig)
	if err != nil {
		return nil, err
	}

	dialerFactory, err := dialer.NewFactory(scaledCtx)
	scaledCtx.Dialer = dialerFactory

	return scaledCtx, nil
}

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")

	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "kube-server.conf")
}

func Run(ctx context.Context) error {
	prepareEnv()

	opt := options.Options
	httpsAddr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))

	if opt.KubeConfig == "" {
		return fmt.Errorf("kube config file must provided")
	}
	kubeConfig, err := k8s.GetConfig(opt.KubeConfig)
	if err != nil {
		return err
	}

	scaledCtx, err := buildScaledContext(ctx, *kubeConfig)
	if err != nil {
		return err
	}

	if err := server.Start(httpsAddr, scaledCtx); err != nil {
		return err
	}
	return nil
}
