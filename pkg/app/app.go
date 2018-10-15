package app

import (
	"context"
	"net"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/util/runtime"

	"yunion.io/x/yunion-kube/pkg/clusterdriver"
	"yunion.io/x/yunion-kube/pkg/clusterdriver/yke"
	"yunion.io/x/yunion-kube/pkg/controllers"
	"yunion.io/x/yunion-kube/pkg/dialer"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/server"
	_ "yunion.io/x/yunion-kube/pkg/tasks"
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

	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "kube-server.conf")
	runtime.ReallyCrash = false
}

func Run(ctx context.Context) error {
	prepareEnv()
	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := cloudcommon.InitApp(&options.Options.Options)
	InitHandlers(app)

	if db.CheckSync(options.Options.AutoSyncTable) {
		err := models.InitDB()
		if err != nil {
			log.Fatalf("Init models error: %v", err)
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

	cloudcommon.InitAuth(&options.Options.Options, func() {
		log.Infof("Auth complete, start controllers.")
		controllers.Start()
	})

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
