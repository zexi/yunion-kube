package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IHostDriver interface {
	GetHostType() string
	CheckAndSetCacheImage(ctx context.Context, host *SHost, storagecache *SStoragecache, task taskman.ITask) error
	RequestPrepareSaveDiskOnHost(ctx context.Context, host *SHost, disk *SDisk, imageId string, task taskman.ITask) error
	RequestSaveUploadImageOnHost(ctx context.Context, host *SHost, disk *SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error
	RequestAllocateDiskOnStorage(ctx context.Context, host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask, content *jsonutils.JSONDict) error
	RequestDeallocateDiskOnHost(host *SHost, storage *SStorage, disk *SDisk, task taskman.ITask) error
	RequestResizeDiskOnHostOnline(host *SHost, storage *SStorage, disk *SDisk, size int64, task taskman.ITask) error
	RequestResizeDiskOnHost(host *SHost, storage *SStorage, disk *SDisk, size int64, task taskman.ITask) error
	RequestDeleteSnapshotsWithStorage(ctx context.Context, host *SHost, snapshot *SSnapshot, task taskman.ITask) error
	RequestResetDisk(ctx context.Context, host *SHost, disk *SDisk, params *jsonutils.JSONDict, task taskman.ITask) error
	RequestCleanUpDiskSnapshots(ctx context.Context, host *SHost, disk *SDisk, params *jsonutils.JSONDict, task taskman.ITask) error
	PrepareConvert(host *SHost, image, raid string, data jsonutils.JSONObject) (*jsonutils.JSONDict, error)
	PrepareUnconvert(host *SHost) error
	FinishUnconvert(ctx context.Context, userCred mcclient.TokenCredential, host *SHost) error
	FinishConvert(userCred mcclient.TokenCredential, host *SHost, guest *SGuest, hostType string) error
	ConvertFailed(host *SHost) error
	GetRaidScheme(host *SHost, raid string) (string, error)
}

var hostDrivers map[string]IHostDriver

func init() {
	hostDrivers = make(map[string]IHostDriver)
}

func RegisterHostDriver(driver IHostDriver) {
	hostDrivers[driver.GetHostType()] = driver
}

func GetHostDriver(hostType string) IHostDriver {
	driver, ok := hostDrivers[hostType]
	if ok {
		return driver
	} else {
		log.Fatalf("Unsupported hostType %s", hostType)
		return nil
	}
}
