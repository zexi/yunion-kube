package machines

import (
	"context"
	"fmt"

	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

var MachineManager *SMachineManager

func init() {
	MachineManager = &SMachineManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SMachine{}, "machines_tbl", "machine", "machines"),
	}
}

type SMachineManager struct {
	db.SVirtualResourceBaseManager
}

type SMachine struct {
	db.SVirtualResourceBase
	// Provider determine which cloud provider this node used, e.g. onecloud, aliyun, aws
	Provider  string `nullable:"false" create:"required" list:"user"`
	ClusterId string `nullable:"false" create:"required" list:"user"`
	Role      string `nullable:"false" create:"required" list:"user"`
	// ResourceType determine which resource type this node used
	ResourceType string `nullable:"false" create:"required" list:"user"`
	// InstanceId related to cloud host or guest id
	InstanceId string `nullable:"true" create:"optional" list:"user"`
	// TODO: cloudprovider
}

func (man *SMachineManager) GetCluster(userCred mcclient.TokenCredential, clusterId string) (*clusters.SCluster, error) {
	obj, err := clusters.ClusterManager.FetchByIdOrName(userCred, clusterId)
	if err != nil {
		return nil, err
	}
	return obj.(*clusters.SCluster), nil
}

func (man *SMachineManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	clusterId, _ := data.GetString("cluster")
	if len(clusterId) == 0 {
		return nil, httperrors.NewInputParameterError("Cluster must specified")
	}
	if cluster, err := man.GetCluster(userCred, clusterId); err != nil {
		return nil, httperrors.NewNotFoundError("Cluster %s not found", clusterId)
	} else {
		clusterId = cluster.GetId()
		data.Set("cluster_id", jsonutils.NewString(clusterId))
		data.Set("provider", jsonutils.NewString(cluster.Provider))
	}

	resourceType := clusters.SetJSONDataDefault(data, "resource_type", string(types.MachineResourceTypeBaremetal))
	if !utils.IsInStringArray(resourceType, []string{string(types.MachineResourceTypeBaremetal)}) {
		return nil, httperrors.NewInputParameterError("Invalid resource type: %q", resourceType)
	}

	if role, _ := data.GetString("role"); !utils.IsInStringArray(role, []string{string(types.RoleTypeControlplane), string(types.RoleTypeNode)}) {
		return nil, httperrors.NewInputParameterError("Invalid role: %q", role)
	}
	// TODO: drivers ValidateCreateData

	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (m *SMachine) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	m.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	if err := m.StartMachineCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartMachineCreateTask error: %v", err)
	}
}

func (m *SMachine) StartMachineCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "MachineCreateTask", m, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SMachine) allowPerformAction(userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.IsOwner(userCred)
}

func (m *SMachine) AllowPerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.allowPerformAction(userCred, query, data)
}

func (m *SMachine) ValidatePrepareCondition(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) error {
	return nil
}

type MachinePrepareData struct {
	Script     string
	InstanceId string
}

func (m *SMachine) PerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := m.ValidatePrepareCondition(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	if m.Status != types.MachineStatusInit {
		return nil, fmt.Errorf("machine status is not init")
	}
	// TODO: task async
	driver := GetDriver(types.ProviderType(m.Provider))
	m.SetStatus(userCred, types.MachineStatusPrepare, "")
	ret, err := driver.PrepareResource(nil, nil)
	if err != nil {
		m.SetStatus(userCred, types.MachineStatusPrepareFail, "")
		return nil, httperrors.NewGeneralError(err)
	}
	m.SetStatus(userCred, types.MachineStatusRunning, "")
	log.Infof("Prepare complete, ret: %v", ret)
	return nil, nil
}

func (m *SMachine) GetGlobalClient() (*clientset.Clientset, error) {
	return clusters.ClusterManager.GetGlobalClient()
}
