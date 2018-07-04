package main

import (
	"context"
	"os"

	"yunion.io/yunioncloud/pkg/cloudcommon"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/app"
	"yunion.io/yunion-kube/pkg/options"
)

func main() {
	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "kube-server.conf")

	err := app.Run(context.Background())
	log.Fatalf("Run err: %v", err)
}
