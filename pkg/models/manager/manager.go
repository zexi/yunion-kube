package manager

import (
	"context"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/models/types"
)

type ICluster interface {
	GetId() string
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
	//SetKubeconfig(kubeconfig string) error
}

type IClusterManager interface {
	IsClusterExists(userCred mcclient.TokenCredential, id string) (ICluster, bool, error)
	FetchClusterByIdOrName(userCred mcclient.TokenCredential, id string) (ICluster, error)
	CreateCluster(ctx context.Context, userCred mcclient.TokenCredential, data types.CreateClusterData) (ICluster, error)
	//GetNonSystemClusters() ([]ICluster, error)
}

type IMachine interface {
	GetId() string
	GetName() string
	IsFirstNode() bool
	GetResourceId() string
	IsControlplane() bool
	IsRunning() bool
	GetPrivateIP() (string, error)
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
	GetStatus() string
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
	SetPrivateIP(address string) error
	GetRole() string
}

type IMachineManager interface {
	FetchMachineByIdOrName(userCred mcclient.TokenCredential, id string) (IMachine, error)
	GetMachines(clusterId string) ([]IMachine, error)
	IsMachineExists(userCred mcclient.TokenCredential, id string) (IMachine, bool, error)
	CreateMachine(ctx context.Context, userCred mcclient.TokenCredential, data *types.CreateMachineData) (IMachine, error)
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
