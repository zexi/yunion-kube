package app

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon"
	app_commmon "yunion.io/x/onecloud/pkg/cloudcommon/app"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/pkg/util/runtime"

	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/initial"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/server"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/yunion-kube/pkg/controllers"
)

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")

	common_options.ParseOptions(&options.Options, os.Args, "kube-server.conf", "k8s")
	runtime.ReallyCrash = false
	helm.InitEnv(options.Options.HelmDataDir)
}

func Run(ctx context.Context) error {
	prepareEnv()
	cloudcommon.InitDB(&options.Options.DBOptions)
	defer cloudcommon.CloseDB()

	app := app_commmon.InitApp(&options.Options.BaseOptions, true)
	InitHandlers(app)

	app_commmon.InitAuth(&options.Options.CommonOptions, func() {})

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

	opt := options.Options
	httpsAddr := net.JoinHostPort(opt.Address, strconv.Itoa(opt.HttpsPort))

	if err := models.ClusterManager.RegisterSystemCluster(); err != nil {
		log.Fatalf("Register system cluster %v", err)
	}
	initial.InitClient()

	cron := cronman.InitCronJobManager(true, options.Options.CronJobWorkerCount)
	cron.AddJobAtIntervalsWithStartRun("StartKubeClusterHealthCheck", time.Minute, models.ClusterManager.ClusterHealthCheckTask, true)
	cron.Start()
	defer cron.Stop()

	if err := server.Start(httpsAddr, app); err != nil {
		return err
	}
	return nil
}

func init() {
	signalutils.SetDumpStackSignal()
	signalutils.StartTrap()
}
