package models

import (
	"context"
	"fmt"

	"yunion.io/yunioncloud/pkg/cloudcommon/db"
	"yunion.io/yunioncloud/pkg/jsonutils"
	"yunion.io/yunioncloud/pkg/log"
	"yunion.io/yunioncloud/pkg/mcclient"

	"yunion.io/yunion-kube/pkg/types/apis"
)

var ClusterManager *SClusterManager

func init() {
	ClusterManager = &SClusterManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SCluster{}, "clusters_tbl", "cluster", "clusters"),
	}
}

type SClusterManager struct {
	db.SVirtualResourceBaseManager
}

func (m *SClusterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return m.SVirtualResourceBaseManager.AllowListItems(ctx, userCred, query)
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SClusterManager) FetchCluster(ident string) *SCluster {
	cluster, err := m.FetchByIdOrName("", ident)
	if err != nil {
		log.Errorf("Fetch cluster %q fail: %v", ident, err)
		return nil
	}
	return cluster.(*SCluster)
}

func (m *SClusterManager) GetCluster(ident string) (*apis.Cluster, error) {
	cluster := m.FetchCluster(ident)
	if cluster == nil {
		return nil, fmt.Errorf("Not found cluster %q", ident)
	}
	return cluster.Cluster()
}

type SCluster struct {
	db.SVirtualResourceBase
	Mode          string               `nullable:"false" create:"required" list:"admin"`
	Spec          jsonutils.JSONObject `nullable:"true" list:"admin"`
	ClusterStatus jsonutils.JSONObject `nullable:"true" list:"admin"`
}

func (c *SCluster) Cluster() (*apis.Cluster, error) {
	spec := apis.ClusterSpec{}
	status := apis.ClusterStatus{}
	if c.Spec != nil {
		c.Spec.Unmarshal(&spec)
	}
	if c.ClusterStatus != nil {
		c.ClusterStatus.Unmarshal(&status)
	}
	return &apis.Cluster{
		Name:   c.Name,
		Spec:   spec,
		Status: status,
	}, nil
}
