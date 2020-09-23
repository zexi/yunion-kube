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
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/pkg/util/runtime"
	"yunion.io/x/pkg/util/signalutils"

	"yunion.io/x/yunion-kube/pkg/api/constants"
	"yunion.io/x/yunion-kube/pkg/controllers"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/initial"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/options"
	_ "yunion.io/x/yunion-kube/pkg/policy"
	"yunion.io/x/yunion-kube/pkg/server"
)

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")

	common_options.ParseOptions(&options.Options, os.Args, "kube-server.conf", constants.ServiceType)
	runtime.ReallyCrash = false
	helm.InitEnv(options.Options.HelmDataDir)
}

func Run(ctx context.Context) error {
	opt := &options.Options
	prepareEnv()
	cloudcommon.InitDB(&opt.DBOptions)
	defer cloudcommon.CloseDB()

	app := app_commmon.InitApp(&opt.BaseOptions, true)
	InitHandlers(app)

	app_commmon.InitAuth(&opt.CommonOptions, func() {})
	common_options.StartOptionManager(opt, opt.ConfigSyncPeriodSeconds, constants.ServiceType, constants.ServiceVersion, options.OnOptionsChange)

	if db.CheckSync(options.Options.AutoSyncTable) {
		for _, initDBFunc := range []func() error{
			models.InitDB,
		} {
			err := initDBFunc()
			if err != nil {
				log.Fatalf("Init models error: %v", err)
			}
		}
	} else {
		log.Fatalf("Fail sync db")
	}

	go func() {
		log.Infof("Auth complete, start controllers.")
		controllers.Start()
	}()

	httpsAddr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))

	cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
	initial.InitClient(cron)
	cron.Start()
	defer cron.Stop()

	if err := models.ClusterManager.RegisterSystemCluster(); err != nil {
		log.Fatalf("Register system cluster %v", err)
	}

	if err := server.Start(httpsAddr, app); err != nil {
		return err
	}
	return nil
}

func init() {
	signalutils.SetDumpStackSignal()
	signalutils.StartTrap()
}
