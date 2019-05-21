package yunion_host

import (
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/machines"
	"yunion.io/x/yunion-kube/pkg/models/types"
	onecloudcli "yunion.io/x/yunion-kube/pkg/utils/onecloud/client"
)

const (
	HostTypeKVM     = "hypervisor"
	HostTypeKubelet = "kubelet"
)

func ValidateResourceType(resType string) error {
	if resType != types.MachineResourceTypeBaremetal {
		return httperrors.NewInputParameterError("Invalid resource type: %q", resType)
	}
	return nil
}

func ValidateHostId(s *mcclient.ClientSession, privateKey string, hostId string) (jsonutils.JSONObject, error) {
	ret, err := cloudmod.Hosts.Get(s, hostId, nil)
	if err != nil {
		return nil, err
	}
	hostType, _ := ret.GetString("host_type")
	hostId, _ = ret.GetString("id")
	if m, err := machines.MachineManager.GetMachineByResourceId(hostId); err != nil {
		return nil, err
	} else if m != nil {
		return nil, httperrors.NewInputParameterError("Machine %s already use host %s", m.GetName(), hostId)
	}
	if !utils.IsInStringArray(hostType, []string{HostTypeKVM, HostTypeKubelet}) {
		return nil, httperrors.NewInputParameterError("Host %q invalid host_type %q", hostId, hostType)
	}
	accessIP, _ := ret.GetString("access_ip")
	if err := RemoteCheckHostEnvironment(accessIP, 22, "root", privateKey); err != nil {
		return nil, httperrors.NewUnsupportOperationError("host %s: %v", accessIP, err.Error())
	}
	return ret, nil
}

func GetV1Cluster(cluster *clusters.SCluster) (*models.SCluster, error) {
	return models.ClusterManager.FetchClusterByIdOrName(nil, cluster.GetName())
}

func GetV1Node(machine *machines.SMachine) (*models.SNode, error) {
	return models.NodeManager.FetchNodeByHostId(machine.ResourceId)
}

func validateCreateMachine(s *mcclient.ClientSession, privateKey string, m *types.CreateMachineData) error {
	if err := machines.ValidateRole(m.Role); err != nil {
		return err
	}
	if err := ValidateResourceType(m.ResourceType); err != nil {
		return err
	}
	if len(m.ResourceId) == 0 {
		return httperrors.NewInputParameterError("ResourceId must provided")
	}
	hostObj, err := ValidateHostId(s, privateKey, m.ResourceId)
	if err != nil {
		return err
	}
	hostName, err := hostObj.GetString("name")
	if err != nil {
		return errors.Wrap(err, "not found host name")
	}
	m.Name = hostName
	return nil
}

func CheckControlplaneExists(cluster *clusters.SCluster) error {
	controlplane, err := cluster.GetRunningControlplaneMachine()
	if err != nil {
		return httperrors.NewInputParameterError("CheckControlplaneExists: %v", err)
	}
	if controlplane == nil {
		return fmt.Errorf("Running controlplane not exists")
	}
	return nil
}

func ValidateCreateMachines(ms []*types.CreateMachineData) error {
	session, err := clusters.ClusterManager.GetSession()
	if err != nil {
		return err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return err
	}

	//ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	//errgrp, _ := errgroup.WithContext(ctx)
	var errgrp errgroup.Group
	for _, m := range ms {
		tmp := m
		errgrp.Go(func() error {
			if err := validateCreateMachine(session, privateKey, tmp); err != nil {
				return err
			}
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	return nil
}

func ValidateClusterCreateData(data *jsonutils.JSONDict) error {
	createData := types.CreateClusterData{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	ms := createData.Machines
	controls, _ := drivers.GetControlplaneMachineDatas("", ms)
	if len(controls) == 0 && createData.Provider != string(types.ProviderTypeSystem) {
		return httperrors.NewInputParameterError("No controlplane nodes")
	}
	session, err := clusters.ClusterManager.GetSession()
	if err != nil {
		return err
	}
	privateKey, err := onecloudcli.GetCloudSSHPrivateKey(session)
	if err != nil {
		return err
	}
	//ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	//errgrp, _ := errgroup.WithContext(ctx)
	var errgrp errgroup.Group
	for _, m := range ms {
		tmp := m
		errgrp.Go(func() error {
			if err := validateCreateMachine(session, privateKey, tmp); err != nil {
				return err
			}
			return nil
		})
	}
	return nil
}

func RemoteCheckHostsEnvironment(hosts []string, privateKey string) error {
	//ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	//errgrp, _ := errgroup.WithContext(ctx)
	var errgrp errgroup.Group
	for _, h := range hosts {
		tmp := h
		errgrp.Go(func() error {
			if err := RemoteCheckHostEnvironment(tmp, 22, "root", privateKey); err != nil {
				return fmt.Errorf("Host %s bad environment: %v", tmp, err)
			}
			return nil
		})
	}
	if err := errgrp.Wait(); err != nil {
		return err
	}
	return nil
}

func RemoteCheckHostEnvironment(host string, port int, username string, privateKey string) error {
	cli, err := ssh.NewClient(host, port, username, "", privateKey)
	if err != nil {
		return fmt.Errorf("create ssh connection: %v", host, err)
	}
	_, err = cli.Run("which docker kubeadm kubelet")
	if err != nil {
		return fmt.Errorf("required binary not exists: %v", err)
	}
	return nil
}
