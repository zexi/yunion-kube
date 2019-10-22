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

	"yunion.io/x/yunion-kube/pkg/controllers"
	"yunion.io/x/yunion-kube/pkg/initial"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/options"
	"yunion.io/x/yunion-kube/pkg/server"
)

func prepareEnv() {
	os.Unsetenv("SSH_AUTH_SOCK")
	os.Unsetenv("SSH_AGENT_PID")
	os.Setenv("DISABLE_HTTP2", "true")

	common_options.ParseOptions(&options.Options, os.Args, "kube-server.conf", "k8s")
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

	app_commmon.InitAuth(&options.Options.CommonOptions, func() {
		log.Infof("Auth complete, start controllers.")
		go func() {
			controllers.Start()
		}()
	})

	initial.InitClient()

	if err := server.Start(httpsAddr, app); err != nil {
		return err
	}
	return nil
}
