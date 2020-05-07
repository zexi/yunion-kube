package models

import "yunion.io/x/yunion-kube/pkg/client"

type SClusterResourceBaseManager struct{}

type SClusterResourceBase struct {
	ClusterId string `width:"128" charset:"ascii" nullable:"true" index:"true" list:"user"`
}

type SNamespaceResourceBaseManager struct{}

type SNamespaceResourceBase struct {
	SClusterResourceBase

	Namespace string `width:"128" charset:"ascii" nullable:"true" index:"true" list:"user"`
}

func (res *SClusterResourceBase) GetCluster() (*SCluster, error) {
	obj, err := ClusterManager.FetchById(res.ClusterId)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (res *SClusterResourceBase) GetClusterClient() (*client.ClusterManager, error) {
	cls, err := res.GetCluster()
	if err != nil {
		return nil, err
	}
	return client.GetManagerByCluster(cls)
}
