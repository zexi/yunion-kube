package config

import (
	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/types/config/dialer"
)

type ScaledContext struct {
	Dialer         dialer.Factory
	ClusterManager *models.SClusterManager
	NodeManager    *models.SNodeManager
}

func NewScaledContext() (*ScaledContext, error) {
	ctx := &ScaledContext{}
	return ctx, nil
}

func (ctx *ScaledContext) SetDialerFactory(factory dialer.Factory) *ScaledContext {
	ctx.Dialer = factory
	return ctx
}

func (ctx *ScaledContext) SetClusterManager(man *models.SClusterManager) *ScaledContext {
	ctx.ClusterManager = man
	return ctx
}

func (ctx *ScaledContext) SetNodeManager(man *models.SNodeManager) *ScaledContext {
	ctx.NodeManager = man
	return ctx
}
