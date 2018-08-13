package main

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/app"
)

func main() {
	err := app.Run(context.Background())
	log.Fatalf("Run err: %v", err)
}
