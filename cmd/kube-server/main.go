package main

import (
	"context"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/app"
)

func main() {
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("Run err: %v", err)
	}
}
