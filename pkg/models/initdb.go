package models

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func InitDB() error {
	for _, manager := range []db.IModelManager{
		ClusterManager,
		RepoManager,
	} {
		err := manager.InitializeData()
		if err != nil {
			return fmt.Errorf("Manager %s InitializeData error: %v", err)
		}
	}
	return nil
}
