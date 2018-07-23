package app

import (
	"context"
	"net"
	"os"
	"strconv"

	"yunion.io/yunioncloud/pkg/appsrv"
	"yunion.io/yunioncloud/pkg/cloudcommon"
	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/util/runtime"

	"yunion.io/yunion-kube/pkg/clusterdriver"
	"yunion.io/yunion-kube/pkg/clusterdriver/yke"
	"yunion.io/yunion-kube/pkg/dialer"
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/server"
	"yunion.io/yunion-kube/pkg/types/config"
	"yunion.io/yunion-kube/pkg/ykedialerfactory"
)

func buildScaledContext(ctx context.Context) (*config.ScaledContext, error) {
	scaledCtx, err := config.NewScaledContext()
	if err != nil {
		return nil, err
	}

	dialerFactory, err := dialer.NewFactory(scaledCtx)
	return scaledCtx.SetDialerFactory(dialerFactory).
		SetClusterManager(models.ClusterManager).
		SetNodeManager(models.NodeManager), nil
}

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")

	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "kube-server.conf")
	runtime.ReallyCrash = false
}

func initCloudApp() *appsrv.Application {
	app := cloudcommon.InitApp(&options.Options.Options)
	InitHandlers(app)

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete!!!")
	})

	return app
}

func Run(ctx context.Context) error {
	prepareEnv()
	app := initCloudApp()
	// must before InitDB?
	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()
	if !db.CheckSync(options.Options.AutoSyncTable) {
		log.Fatalf("Fail sync db")
	}

	opt := options.Options
	httpsAddr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))

	scaledCtx, err := buildScaledContext(ctx)
	if err != nil {
		return err
	}

	go RegisterDriver(scaledCtx)

	if err := server.Start(httpsAddr, scaledCtx, app); err != nil {
		return err
	}
	return nil
}

func RegisterDriver(scaledCtx *config.ScaledContext) {
	local := &ykedialerfactory.YKEDialerFactory{
		Factory: scaledCtx.Dialer,
	}
	docker := &ykedialerfactory.YKEDialerFactory{
		Factory: scaledCtx.Dialer,
		Docker:  true,
	}
	driver := clusterdriver.Drivers["yke"]
	ykeDriver := driver.(*yke.Driver)
	ykeDriver.DockerDialer = docker.Build
	ykeDriver.LocalDialer = local.Build
	ykeDriver.WrapTransportFactory = docker.WrapTransport
}
