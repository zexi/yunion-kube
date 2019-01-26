package manager

import (
	//"context"

	"yunion.io/x/log"
	//"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

type ICluster interface {
}

type IClusterManager interface {
	//CreateCluster(ctx context.Context, ) (ICluster, error)
}

type IMachine interface {
	IsControlplane() bool
	IsRunning() bool
	GetKubeConfig() (string, error)
}

type IMachineManager interface {
	GetMachines(clusterId string) ([]IMachine, error)
}

var (
	clusterManager IClusterManager
	machineManager IMachineManager
)

func RegisterClusterManager(man IClusterManager) {
	if clusterManager != nil {
		log.Fatalf("ClusterManager already registered")
	}
	clusterManager = man
}

func RegisterMachineManager(man IMachineManager) {
	if machineManager != nil {
		log.Fatalf("MachineManager already registered")
	}
	machineManager = man
}

func ClusterManager() IClusterManager {
	return clusterManager
}

func MachineManager() IMachineManager {
	return machineManager
}
