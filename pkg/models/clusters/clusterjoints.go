package clusters

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SClusterJointsManager struct {
	db.SVirtualJointResourceBaseManager
}

func NewClusterJointsManager(dt interface{}, tableName string, keyword string, keywordPlural string, slave db.IVirtualModelManager) SClusterJointsManager {
	return SClusterJointsManager{
		SVirtualJointResourceBaseManager: db.NewVirtualJointResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
			ClusterManager,
			slave,
		),
	}
}

type SClusterJointsBase struct {
	db.SVirtualJointResourceBase

	ClusterId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func (s *SClusterJointsBase) GetCluster() *SCluster {
	obj, _ := ClusterManager.FetchById(s.ClusterId)
	return obj.(*SCluster)
}
