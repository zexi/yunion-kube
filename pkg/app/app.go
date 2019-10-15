package app

import (
	"context"
	"net"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_commmon "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/pkg/util/runtime"

	"yunion.io/x/yunion-kube/pkg/clusterdriver"
	"yunion.io/x/yunion-kube/pkg/clusterdriver/yke"
	"yunion.io/x/yunion-kube/pkg/controllers"
	"yunion.io/x/yunion-kube/pkg/dialer"
	"yunion.io/x/yunion-kube/pkg/initial"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/server"
	"yunion.io/x/yunion-kube/pkg/types/config"
	"yunion.io/x/yunion-kube/pkg/ykedialerfactory"
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

	common_options.ParseOptions(&options.Options, os.Args, "kube-server.conf", "k8s")
	// TODO: support rbac
	options.Options.EnableRbac = false
	runtime.ReallyCrash = false
}

func Run(ctx context.Context) error {
	prepareEnv()
	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := app_commmon.InitApp(&options.Options.BaseOptions, true)
	InitHandlers(app)

	if db.CheckSync(options.Options.AutoSyncTable) {
		for _, initDBFunc := range []func() error{
			models.InitDB,
			clusters.InitDB,
		} {
			err := initDBFunc()
			if err != nil {
				log.Fatalf("Init models error: %v", err)
			}
		}
	} else {
		log.Fatalf("Fail sync db")
	}

	opt := options.Options
	httpsAddr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))

	scaledCtx, err := buildScaledContext(ctx)
	if err != nil {
		return err
	}

	go RegisterDriver(scaledCtx)

	app_commmon.InitAuth(&options.Options.CommonOptions, func() {
		log.Infof("Auth complete, start controllers.")
		go func() {
			controllers.Start()
		}()
		if err := models.ClusterManager.StartMigrate(); err != nil {
			log.Errorf("Migrate cluster error: %v", err)
		}
	})

	initial.InitClient()

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
	ykeDriver.DialerFactory = scaledCtx.Dialer
	ykeDriver.DockerDialer = docker.Build
	ykeDriver.LocalDialer = local.Build
	ykeDriver.WrapTransportFactory = docker.WrapTransport
}
