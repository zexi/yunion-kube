package machines

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

var MachineManager *SMachineManager

func init() {
	MachineManager = &SMachineManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SMachine{}, "machines_tbl", "kubemachine", "kubemachines"),
	}
	manager.RegisterMachineManager(MachineManager)
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
	// ResourceId related to cloud host or guest id
	ResourceId string `nullable:"true" create:"optional" list:"user"`
	// TODO: cloudprovider
	// FirstNode determine machine is first controlplane
	FirstNode tristate.TriState `nullable:"true" list:"user"`
}

func (man *SMachineManager) GetCluster(userCred mcclient.TokenCredential, clusterId string) (*clusters.SCluster, error) {
	obj, err := clusters.ClusterManager.FetchByIdOrName(userCred, clusterId)
	if err != nil {
		return nil, err
	}
	return obj.(*clusters.SCluster), nil
}

func (man *SMachineManager) GetSession() (*mcclient.ClientSession, error) {
	return models.GetAdminSession()
}

func (man *SMachineManager) GetMachines(clusterId string) ([]manager.IMachine, error) {
	machines := man.Query().SubQuery()
	q := machines.Query().Filter(sqlchemy.Equals(machines.Field("cluster_id"), clusterId))
	objs := make([]SMachine, 0)
	err := db.FetchModelObjects(man, q, &objs)
	if err != nil {
		return nil, err
	}
	ret := make([]manager.IMachine, len(objs))
	for i := range objs {
		ret[i] = &objs[i]
	}
	return ret, nil
}

func (man *SMachineManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	clusterId, _ := data.GetString("cluster")
	if len(clusterId) == 0 {
		return nil, httperrors.NewInputParameterError("Cluster must specified")
	}
	cluster, err := man.GetCluster(userCred, clusterId)
	if err != nil {
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
	driver := GetDriver(types.ProviderType(cluster.Provider))
	session, err := man.GetSession()
	if err != nil {
		return nil, err
	}
	if err := driver.ValidateCreateData(session, userCred, ownerProjId, query, data); err != nil {
		return nil, err
	}

	machine := types.Machine{}
	data.Unmarshal(&machine)
	if err := cluster.ValidateAddMachine(&machine); err != nil {
		return nil, err
	}

	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (m *SMachine) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	m.SVirtualResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)
	if err := m.StartMachineCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartMachineCreateTask error: %v", err)
	}
}

func (m *SMachine) StartMachineCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := m.SetStatus(userCred, types.MachineStatusCreating, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "MachineCreateTask", m, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SMachine) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := m.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	return m.moreExtraInfo(extra)
}

func (m *SMachine) moreExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	cluster, _ := m.GetCluster()
	if cluster != nil {
		extra.Add(jsonutils.NewString(cluster.Name), "cluster")
		extra.Add(jsonutils.NewString(cluster.GetNamespace()), "namespace")
	}
	return extra
}

func (m *SMachine) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := m.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}

	return m.moreExtraInfo(extra), nil
}

func (m *SMachine) allowPerformAction(userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.IsOwner(userCred)
}

func (m *SMachine) AllowPerformRecreate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.allowPerformAction(userCred, query, data)
}

func (m *SMachine) PerformRecreate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if m.Status != types.MachineStatusCreateFail {
		return nil, httperrors.NewForbiddenError("Status is %s", m.SetStatus)
	}
	if err := m.StartMachineCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (m *SMachine) AllowPerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.allowPerformAction(userCred, query, data)
}

func (m *SMachine) ValidatePrepareCondition(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) error {
	return nil
}

type MachinePrepareData struct {
	FirstNode bool   `json:"firstNode"`
	Role      string `json:"role"`

	CAKeyPair           *types.KeyPair `json:"caKeyPair"`
	EtcdCAKeyPair       *types.KeyPair `json:"etcdCAKeyPair"`
	FrontProxyCAKeyPair *types.KeyPair `json:"frontProxyCAKeyPair"`
	SAKeyPair           *types.KeyPair `json:"saKeyPair"`
	BootstrapToken      string         `json:"bootstrapToken"`
	ELBAddress          string         `json:"elbAddress"`

	InstanceId string `json:"-"`
	PrivateIP  string `json:"-"`
}

func (m *SMachine) StartPrepareTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := m.SetStatus(userCred, types.MachineStatusPrepare, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "MachinePrepareTask", m, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SMachine) PerformPrepare(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := m.ValidatePrepareCondition(ctx, userCred, query, data); err != nil {
		return nil, err
	}
	if utils.IsInStringArray(m.Status, []string{types.MachineStatusRunning, types.MachineStatusDeleting}) {
		return nil, fmt.Errorf("machine status is %s", m.Status)
	}

	return nil, m.StartPrepareTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (m *SMachine) GetGlobalClient() (*clientset.Clientset, error) {
	return clusters.ClusterManager.GetGlobalClient()
}

func (m *SMachine) GetCluster() (*clusters.SCluster, error) {
	return clusters.ClusterManager.GetCluster(m.ClusterId)
}

func (m *SMachine) GetNamespace() string {
	cluster, err := m.GetCluster()
	if err != nil {
		log.Errorf("GetCluster: %v", err)
		return ""
	}
	return cluster.GetNamespace()
}

func (m *SMachine) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Machine delete do nothing")
	return nil
}

func (m *SMachine) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return m.SVirtualResourceBase.Delete(ctx, userCred)
}

func (m *SMachine) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	m.SetStatus(userCred, types.MachineStatusDeleting, "")
	return m.startRemoveMachine(ctx, userCred)
}

func (m *SMachine) startRemoveMachine(ctx context.Context, userCred mcclient.TokenCredential) error {
	cli, err := m.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	if err := cli.ClusterV1alpha1().Machines(m.GetNamespace()).Delete(m.Name, &v1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

func (m *SMachine) AllowPerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.allowPerformAction(userCred, query, data)
}

func (m *SMachine) PerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, m.StartTerminateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (m *SMachine) StartTerminateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := m.SetStatus(userCred, types.MachineStatusTerminating, ""); err != nil {
		return err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "MachineTerminateTask", m, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (m *SMachine) GetPrivateIP() (string, error) {
	driver := m.GetDriver()
	session, err := MachineManager.GetSession()
	if err != nil {
		return "", err
	}
	return driver.GetPrivateIP(session, m.ResourceId)
}

func (m *SMachine) GetDriver() IMachineDriver {
	return GetDriver(types.ProviderType(m.Provider))
}

func (m *SMachine) GetRole() string {
	return m.Role
}

func (m *SMachine) IsControlplane() bool {
	return m.GetRole() == types.RoleTypeControlplane
}

func (m *SMachine) IsRunning() bool {
	return m.Status == types.MachineStatusRunning
}

func (m *SMachine) IsFirstNode() bool {
	return m.FirstNode.Bool()
}

func (m *SMachine) GetKubeConfig() (string, error) {
	driver := m.GetDriver()
	session, err := MachineManager.GetSession()
	if err != nil {
		return "", err
	}
	return driver.GetKubeConfig(session, m)
}
