package main

import (
	"context"

	"github.com/yunionio/log"

	"yunion.io/yunion-kube/pkg/app"
)

func main() {
	err := app.Run(context.Background())
	log.Fatalf("Run err: %v", err)
}
