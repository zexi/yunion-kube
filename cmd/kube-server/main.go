package main

import (
	"os"

	"yunion.io/yunioncloud/pkg/cloudcommon"
	"yunion.io/yunioncloud/pkg/log"

	"yunion.io/yunion-kube/pkg/options"
	"yunion.io/yunion-kube/pkg/server"
)

func main() {
	cloudcommon.ParseOptions(&options.Options, &options.Options.Options, os.Args, "kube-server.conf")

	err := server.Run()
	log.Fatalf("Run err: %v", err)
}
