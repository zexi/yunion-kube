package storageclass

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var StorageClassManager *SStorageClassManager

type SStorageClassManager struct {
	*resources.SResourceBaseManager
}

func init() {
	StorageClassManager = &SStorageClassManager{
		SResourceBaseManager: resources.NewResourceBaseManager("storageclass", "storageclasses"),
	}
}
