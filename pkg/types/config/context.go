package config

import (
	"k8s.io/client-go/rest"

	"yunion.io/yunion-kube/pkg/types/config/dialer"
)

type ScaledContext struct {
	RESTConfig rest.Config
	Dialer     dialer.Factory
}

func NewScaledContext(config rest.Config) (*ScaledContext, error) {
	//var err error

	ctx := &ScaledContext{
		RESTConfig: config,
	}
	return ctx, nil
}
