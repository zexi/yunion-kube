package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	ocapis "yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/wait"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

var MachineManager *SMachineManager

func init() {
	MachineManager = &SMachineManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SMachine{}, "machines_tbl", "kubemachine", "kubemachines"),
	}
	manager.RegisterMachineManager(MachineManager)
	MachineManager.SetVirtualObject(MachineManager)
}

type SMachineManager struct {
	db.SVirtualResourceBaseManager
}

type SMachine struct {
	db.SVirtualResourceBase
	// Provider determine which cloud provider this node used, e.g. onecloud, aliyun, aws
	Provider  string `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	ClusterId string `width:"128" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Role      string `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	// ResourceType determine which resource type this node used
	ResourceType string `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	// ResourceId related to cloud host or guest id
	ResourceId string `width:"128" charset:"ascii" nullable:"true" create:"optional" list:"user"`
	// TODO: cloudprovider
	// FirstNode determine machine is first controlplane
	FirstNode tristate.TriState `nullable:"true" create:"required" list:"user"`

	// Private IP address
	Address string `width:"16" charset:"ascii" nullable:"true" list:"user"`

	Hypervisor string `width:"16" charset:"ascii" nullable:"true" list:"user" create:"optional"`
}

func (man *SMachineManager) GetCluster(userCred mcclient.TokenCredential, clusterId string) (*SCluster, error) {
	obj, err := ClusterManager.FetchByIdOrName(userCred, clusterId)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (man *SMachineManager) GetSession() (*mcclient.ClientSession, error) {
	return GetAdminSession()
}

func (man *SMachineManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}

	var sq *sqlchemy.SSubQuery
	if cluster, _ := query.GetString("cluster"); len(cluster) > 0 {
		clusters := ClusterManager.Query().SubQuery()
		sq = clusters.Query(clusters.Field("id")).
			Filter(sqlchemy.OR(
				sqlchemy.Equals(clusters.Field("name"), cluster),
				sqlchemy.Equals(clusters.Field("id"), cluster))).SubQuery()
	}
	if sq != nil {
		q = q.In("cluster_id", sq)
	}
	return q, nil
}

func (man *SMachineManager) GetMachines(clusterId string) ([]manager.IMachine, error) {
	machines := man.Query().SubQuery()
	q := machines.Query().Filter(sqlchemy.Equals(machines.Field("cluster_id"), clusterId))
	objs := make([]SMachine, 0)
	err := db.FetchModelObjects(man, q, &objs)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ret := make([]manager.IMachine, len(objs))
	for i := range objs {
		ret[i] = &objs[i]
	}
	return ret, nil
}

func (man *SMachineManager) IsMachineExists(userCred mcclient.TokenCredential, id string) (manager.IMachine, bool, error) {
	obj, err := man.FetchByIdOrName(userCred, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, err
	}
	return obj.(*SMachine), true, nil
}

func (man *SMachineManager) CreateMachineNoHook(ctx context.Context, userCred mcclient.TokenCredential, data *apis.CreateMachineData) (manager.IMachine, error) {
	input := jsonutils.Marshal(data)
	obj, err := db.DoCreate(man, ctx, userCred, nil, input, userCred)
	if err != nil {
		return nil, err
	}
	m := obj.(*SMachine)
	err = m.SetMetadata(ctx, apis.MachineMetadataCreateParams, input.String(), userCred)
	return m, err
}

func (m *SMachine) GetCreateInput(userCred mcclient.TokenCredential) (*apis.CreateMachineData, error) {
	input := new(apis.CreateMachineData)
	ret := m.GetMetadataJson(apis.MachineMetadataCreateParams, userCred)
	if ret == nil {
		return nil, errors.Errorf("Not found %s in metadata", apis.MachineMetadataCreateParams)
	}
	err := ret.Unmarshal(input)
	return input, err
}

func (man *SMachineManager) CreateMachine(ctx context.Context, userCred mcclient.TokenCredential, data *apis.CreateMachineData) (manager.IMachine, error) {
	obj, err := man.CreateMachineNoHook(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	m := obj.(*SMachine)
	input := jsonutils.Marshal(data)
	func() {
		lockman.LockObject(ctx, m)
		defer lockman.ReleaseObject(ctx, m)
		m.PostCreate(ctx, userCred, userCred, nil, input)
	}()
	return obj.(*SMachine), nil
}

func (man *SMachineManager) GetClusterControlplaneMachines(clusterId string) ([]*SMachine, error) {
	machines, err := man.GetClusterMachines(clusterId)
	if err != nil {
		return nil, err
	}
	ret := make([]*SMachine, 0)
	for _, m := range machines {
		if m.Role == string(apis.RoleTypeControlplane) {
			ret = append(ret, m)
		}
	}
	return ret, nil
}

func (man *SMachineManager) GetClusterMachines(clusterId string) ([]*SMachine, error) {
	machines := MachineManager.Query().SubQuery()
	q := machines.Query().Filter(sqlchemy.Equals(machines.Field("cluster_id"), clusterId))
	objs := make([]SMachine, 0)
	err := db.FetchModelObjects(MachineManager, q, &objs)
	if err != nil {
		return nil, err
	}
	return ConvertPtrMachines(objs), nil
}

func (man *SMachineManager) GetMachineByResourceId(resId string) (*SMachine, error) {
	machines := MachineManager.Query().SubQuery()
	q := machines.Query().Filter(sqlchemy.Equals(machines.Field("resource_id"), resId))
	m := SMachine{}
	err := q.First(&m)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Errorf("Get machine by resource_id: %v", err)
		return nil, err
	}
	return &m, nil
}

func ConvertPtrMachines(objs []SMachine) []*SMachine {
	ret := make([]*SMachine, len(objs))
	for i, obj := range objs {
		temp := obj
		ret[i] = &temp
	}
	return ret
}

func (man *SMachineManager) FetchMachineById(id string) (*SMachine, error) {
	obj, err := man.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SMachine), nil
}

func (man *SMachineManager) FetchMachineByIdOrName(userCred mcclient.TokenCredential, id string) (manager.IMachine, error) {
	m, err := man.FetchByIdOrName(userCred, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, httperrors.NewNotFoundError("Machine %s", id)
		}
		return nil, err
	}
	return m.(*SMachine), nil
}

func ValidateRole(role string) error {
	if !utils.IsInStringArray(role, []string{
		string(apis.RoleTypeControlplane),
		string(apis.RoleTypeNode),
	}) {
		return httperrors.NewInputParameterError("Invalid role: %q", role)
	}
	return nil
}

func (m *SMachineManager) ValidateResourceType(resType string) error {
	if !utils.IsInStringArray(resType, []string{
		string(apis.MachineResourceTypeVm),
		string(apis.MachineResourceTypeBaremetal),
	}) {
		return httperrors.NewInputParameterError("Invalid machine resource type: %q", resType)
	}
	return nil
}

func (man *SMachineManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	clusterId := jsonutils.GetAnyString(data, []string{"cluster", "cluster_id"})
	if len(clusterId) == 0 {
		return nil, httperrors.NewInputParameterError("Cluster must specified")
	}
	cluster, err := man.GetCluster(userCred, clusterId)
	if err != nil {
		return nil, httperrors.NewNotFoundError("Cluster %s not found", clusterId)
	}
	clusterId = cluster.GetId()
	data.Set("cluster_id", jsonutils.NewString(clusterId))
	data.Set("provider", jsonutils.NewString(cluster.Provider))

	resourceType := SetJSONDataDefault(data, "resource_type", string(apis.MachineResourceTypeBaremetal))
	if err := man.ValidateResourceType(resourceType); err != nil {
		return nil, err
	}

	role, _ := data.GetString("role")
	if err := ValidateRole(role); err != nil {
		return nil, err
	}

	clusterMachines, err := man.GetClusterMachines(clusterId)
	if err != nil {
		return nil, err
	}
	if len(clusterMachines) == 0 && role != string(apis.RoleTypeControlplane) {
		return nil, httperrors.NewInputParameterError("First machine's role must %s", apis.RoleTypeControlplane)
	}
	firstNode := jsonutils.JSONFalse
	if len(clusterMachines) == 0 {
		firstNode = jsonutils.JSONTrue
	}
	data.Set("first_node", firstNode)

	machine := new(apis.CreateMachineData)
	data.Unmarshal(machine)
	ms, err := cluster.ValidateAddMachines(ctx, userCred, []apis.CreateMachineData{*machine})
	if err != nil {
		return nil, err
	}
	data = jsonutils.Marshal(ms[0]).(*jsonutils.JSONDict)

	input := ocapis.VirtualResourceCreateInput{}
	data.Unmarshal(&input)

	if _, err := man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input); err != nil {
		return nil, err
	}
	return data, nil
}

func (m *SMachine) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	m.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := m.StartMachineCreateTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("StartMachineCreateTask error: %v", err)
	}
}

func (m *SMachine) StartMachineCreateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := m.SetStatus(userCred, apis.MachineStatusCreating, ""); err != nil {
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

func (m *SMachine) GetK8sModelNode(cluster *SCluster) (*jsonutils.JSONDict, error) {
	cm, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "client.GetManagerByCluster")
	}
	return model.GetK8SModelObject(cm, apis.KindNameNode, m.Name)
}

func (m *SMachineManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []db.IModel,
	fields stringutils2.SSortedStrings,
) []*jsonutils.JSONDict {
	rows := make([]*jsonutils.JSONDict, len(objs))
	virtRows := m.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields)
	for i := range objs {
		rows[i] = jsonutils.Marshal(virtRows[i]).(*jsonutils.JSONDict)
		rows[i] = objs[i].(*SMachine).moreExtraInfo(rows[i])
	}
	return rows
}

func (m *SMachine) moreExtraInfo(extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	cluster, _ := m.GetCluster()
	if cluster != nil {
		extra.Add(jsonutils.NewString(cluster.Name), "cluster")

		if nodeInfo, err := m.GetK8sModelNode(cluster); err == nil {
			extra.Set("machine_node", nodeInfo)
		} else {
			log.Errorf("fetch k8s model node failed %s", err)
		}
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
	if m.Status != apis.MachineStatusCreateFail {
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

func (m *SMachine) StartPrepareTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := m.SetStatus(userCred, apis.MachineStatusPrepare, ""); err != nil {
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
	if utils.IsInStringArray(m.Status, []string{apis.MachineStatusRunning, apis.MachineStatusDeleting}) {
		return nil, fmt.Errorf("machine status is %s", m.Status)
	}

	return nil, m.StartPrepareTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

func (m *SMachine) GetCluster() (*SCluster, error) {
	return ClusterManager.GetCluster(m.ClusterId)
}

func (m *SMachine) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Machine delete do nothing")
	return nil
}

func (m *SMachine) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return m.SVirtualResourceBase.Delete(ctx, userCred)
}

func (m *SMachine) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	cluster, err := m.GetCluster()
	if err != nil {
		return err
	}
	if err := m.GetDriver().ValidateDeleteCondition(ctx, userCred, cluster, m); err != nil {
		return err
	}
	deleteData := jsonutils.NewDict()
	objs := jsonutils.NewArray()
	objs.Add(jsonutils.NewString(m.GetId()))
	deleteData.Add(objs, "machines")
	return cluster.StartDeleteMachinesTask(ctx, userCred, []manager.IMachine{m}, deleteData, "")
}

func (man *SMachineManager) StartMachineBatchDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, items []db.IStandaloneModel, data *jsonutils.JSONDict, parentTaskId string) error {
	return RunBatchTask(ctx, items, userCred, data, "MachineBatchDeleteTask", parentTaskId)
}

func (m *SMachine) AllowPerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return m.allowPerformAction(userCred, query, data)
}

func (m *SMachine) PerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, m.StartTerminateTask(ctx, userCred, data.(*jsonutils.JSONDict), "")
}

// StartTerminateTask invoke by MachineBatchDeleteTask
func (m *SMachine) StartTerminateTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	if err := m.SetStatus(userCred, apis.MachineStatusTerminating, ""); err != nil {
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
	if len(m.Address) != 0 {
		return m.Address, nil
	}
	driver := m.GetDriver()
	session, err := MachineManager.GetSession()
	if err != nil {
		return "", err
	}
	addr, err := driver.GetPrivateIP(session, m.ResourceId)
	if err != nil {
		return "", err
	}
	return addr, m.SetPrivateIP(addr)
}

func (m *SMachine) SetPrivateIP(address string) error {
	_, err := netutils.NewIPV4Addr(address)
	if err != nil {
		return err
	}
	_, err = m.GetModelManager().TableSpec().Update(m, func() error {
		m.Address = address
		return nil
	})
	return err
}

func (m *SMachine) SetHypervisor(hypervisor string) error {
	if _, err := m.GetModelManager().TableSpec().Update(m, func() error {
		m.Hypervisor = hypervisor
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (m *SMachine) GetDriver() IMachineDriver {
	return GetMachineDriver(apis.ProviderType(m.Provider), apis.MachineResourceType(m.ResourceType))
}

func (m *SMachine) GetRole() string {
	return m.Role
}

func (m *SMachine) IsControlplane() bool {
	return m.GetRole() == apis.RoleTypeControlplane
}

func (m *SMachine) IsRunning() bool {
	return m.Status == apis.MachineStatusRunning
}

func (m *SMachine) IsFirstNode() bool {
	return m.FirstNode.Bool()
}

func WaitMachineRunning(machine *SMachine) error {
	interval := 30 * time.Second
	timeout := 15 * time.Minute
	return wait.Poll(interval, timeout, func() (bool, error) {
		machine, err := MachineManager.FetchMachineById(machine.GetId())
		if err != nil {
			return false, err
		}
		if machine.Status == apis.MachineStatusRunning {
			return true, nil
		}
		if utils.IsInStringArray(machine.Status, []string{apis.MachineStatusPrepare, apis.MachineStatusCreating}) {
			return false, nil
		}
		return false, fmt.Errorf("Machine %s status is %s", machine.GetName(), machine.Status)
	})
}

func WaitMachineDelete(machine *SMachine) error {
	interval := 30 * time.Second
	timeout := 15 * time.Minute
	return wait.Poll(interval, timeout, func() (bool, error) {
		m, exists, err := MachineManager.IsMachineExists(nil, machine.GetId())
		if err != nil {
			return false, err
		}
		if !exists {
			return true, nil
		}
		machine := m.(*SMachine)
		if utils.IsInStringArray(machine.Status, []string{apis.MachineStatusDeleting, apis.MachineStatusTerminating}) {
			return false, nil
		}
		return false, fmt.Errorf("Machine %s status is %s", machine.GetName(), machine.Status)
	})
}

func (m *SMachine) GetResourceId() string {
	return m.ResourceId
}

func (m *SMachine) GetStatus() string {
	return m.Status
}

func (m *SMachine) SetResourceId(id string) error {
	_, err := db.Update(m, func() error {
		m.ResourceId = id
		return nil
	})
	return err
}
