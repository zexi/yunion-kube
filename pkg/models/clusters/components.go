package clusters

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/drivers"
	k8sutil "yunion.io/x/yunion-kube/pkg/k8s/util"
)

var (
	ComponentManager *SComponentManager
)

func init() {
	ComponentManager = NewComponentManager(
		SComponent{},
		"kubecomponent",
		"kubecomponents")
}

func NewComponentManager(dt interface{}, keyword, keywordPlural string) *SComponentManager {
	man := &SComponentManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt, "components_tbl",
			keyword, keywordPlural),
	}
	man.SetVirtualObject(man)
	man.driverManager = drivers.NewDriverManager("")
	return man
}

type SComponentManager struct {
	db.SVirtualResourceBaseManager
	driverManager *drivers.DriverManager
}

type SComponent struct {
	db.SVirtualResourceBase

	Enabled tristate.TriState `nullable:"false" default:"false" list:"user" create:"optional"`

	Type     string               `width:"36" charset:"ascii" nullable:"false" create:"required" list:"user"`
	Settings jsonutils.JSONObject `nullable:"false" list:"user"`
}

type IComponentDriver interface {
	GetType() string
	ValidateCreateData(input *apis.ComponentCreateInput) error
	ValidateUpdateData(input *apis.ComponentUpdateInput) error
	GetCreateSettings(input *apis.ComponentCreateInput) (*apis.ComponentSettings, error)
	GetUpdateSettings(oldSetting *apis.ComponentSettings, input *apis.ComponentUpdateInput) (*apis.ComponentSettings, error)
	PostCreate(cluster *SCluster, obj *SComponent) error
	DoEnable(cluster *SCluster, settings *apis.ComponentSettings) error
	DoDisable(cluster *SCluster, settings *apis.ComponentSettings) error
	FetchStatus(cluster *SCluster, c *SComponent, status *apis.ComponentsStatus) error
}

type baseComponentDriver struct{}

func (m baseComponentDriver) InitStatus(comp *SComponent, out *apis.ComponentStatus) {
	if comp == nil {
		out.Created = false
		out.Enabled = false
		return
	}
	out.Id = comp.GetId()
	out.Created = true
	out.Enabled = comp.Enabled.Bool()
}

func (m *SComponentManager) RegisterDriver(drv IComponentDriver) {
	if err := m.driverManager.Register(drv, drv.GetType()); err != nil {
		panic(errors.Wrapf(err, "component register driver %s", drv.GetType()))
	}
}

func (m *SComponentManager) GetDriver(cType string) (IComponentDriver, error) {
	drv, err := m.driverManager.Get(cType)
	if err != nil {
		if errors.Cause(err) == drivers.ErrDriverNotFound {
			return nil, httperrors.NewNotFoundError("component get by type %s", cType)
		}
		return nil, err
	}
	return drv.(IComponentDriver), nil
}

func (m *SComponentManager) GetDrivers() []IComponentDriver {
	drvs := m.driverManager.GetDrivers()
	ret := make([]IComponentDriver, 0)
	for _, drv := range drvs {
		ret = append(ret, drv.(IComponentDriver))
	}
	return ret
}

func (m *SComponentManager) CreateByCluster(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	input *apis.ComponentCreateInput) (*SComponent, error) {
	if input.Name == "" {
		newName, err := db.GenerateName(m, userCred, fmt.Sprintf("%s-%s", cluster.GetName(), input.Type))
		if err != nil {
			return nil, errors.Wrap(err, "generate component name")
		}
		input.Name = newName
	}
	// 1. create component db record
	obj, err := db.DoCreate(m, ctx, userCred, nil, input.JSON(input), userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "create cluster %s component", cluster.Name)
	}

	// 2. add joint record
	cs := new(SClusterComponent)
	cs.ClusterId = cluster.GetId()
	cs.ComponentId = obj.GetId()
	if err := cs.DoSave(); err != nil {
		return nil, errors.Wrap(err, "create cluster component joint model")
	}

	func() {
		lockman.LockObject(ctx, obj)
		defer lockman.ReleaseObject(ctx, obj)
		obj.PostCreate(ctx, userCred, userCred, nil, nil)
	}()
	return obj.(*SComponent), nil
}

func (m *SComponentManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	_ jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	input := new(apis.ComponentCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return nil, err
	}
	drv, err := m.GetDriver(input.Type)
	if err != nil {
		return nil, err
	}
	if err := drv.ValidateCreateData(input); err != nil {
		return nil, err
	}
	return input.JSON(input), nil
}

func (m *SComponent) CustomizeCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) error {
	input := new(apis.ComponentCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return err
	}
	drv, err := ComponentManager.GetDriver(input.Type)
	if err != nil {
		return err
	}
	settings, err := drv.GetCreateSettings(input)
	if err != nil {
		return errors.Wrapf(err, "get component %s settings", input.Type)
	}
	m.Settings = jsonutils.Marshal(settings)
	return nil
}

func (m *SComponent) GetCluster() (*SCluster, error) {
	result := make([]SCluster, 0)
	query := ClusterManager.Query()
	clustercomponents := ClusterComponentManager.Query().SubQuery()
	q := query.Join(clustercomponents, sqlchemy.AND(
		sqlchemy.Equals(clustercomponents.Field("cluster_id"), query.Field("id")))).
		Filter(sqlchemy.Equals(clustercomponents.Field("component_id"), m.GetId()))
	err := db.FetchModelObjects(ClusterManager, q, &result)
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("Not found cluster by component %s", m.GetId())
	}
	if len(result) != 1 {
		return nil, httperrors.NewDuplicateResourceError("Found %s cluster by component %s", len(result), m.GetId())
	}
	return &result[0], nil
}

func (m *SComponent) GetDriver() (IComponentDriver, error) {
	return ComponentManager.GetDriver(m.Type)
}

func (m *SComponent) PostCreate(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) {
	drv, err := m.GetDriver()
	if err != nil {
		log.Errorf("Get driver by type: %s", m.Type)
		return
	}
	cluster, err := m.GetCluster()
	if err != nil {
		log.Errorf("Get component %s cluster: %v", m.Type, err)
		return
	}
	if err := drv.PostCreate(cluster, m); err != nil {
		log.Errorf("Driver do post create: %s", err)
		return
	}
}

func (m *SComponent) SetEnabled(enabled bool) error {
	_, err := db.Update(m, func() error {
		if enabled {
			m.Enabled = tristate.True
		} else {
			m.Enabled = tristate.False
		}
		return nil
	})
	return err
}

func (m *SComponent) AllowPerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsProjectAllowPerform(userCred, m, "enable")
}

func (m *SComponent) PerformEnable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cluster, err := m.GetCluster()
	if err != nil {
		return nil, err
	}
	return nil, m.DoEnable(cluster)
}

func (m *SComponent) DoEnable(cluster *SCluster) error {
	if err := m.SetEnabled(true); err != nil {
		return err
	}
	drv, err := m.GetDriver()
	if err != nil {
		return err
	}
	settings, err := m.GetSettings()
	if err != nil {
		return err
	}
	return drv.DoEnable(cluster, settings)
}

func (m *SComponent) AllowPerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsProjectAllowPerform(userCred, m, "disable")
}

func (m *SComponent) PerformDisable(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	cluster, err := m.GetCluster()
	if err != nil {
		return nil, err
	}
	return nil, m.DoDisable(cluster)
}

func (m *SComponentManager) EnsureNamespace(cluster *SCluster, namespace string) error {
	k8sMan, err := client.GetManagerByCluster(cluster)
	if err != nil {
		return errors.Wrap(err, "get cluster k8s manager")
	}
	lister := k8sMan.GetIndexer().NamespaceLister()
	cli, err := cluster.GetK8sClient()
	if err != nil {
		return errors.Wrap(err, "get cluster k8s client")
	}
	return k8sutil.EnsureNamespace(lister, cli, namespace)
}

func (m *SComponent) DeleteWithJoint(ctx context.Context, userCred mcclient.TokenCredential) error {
	cs, err := ClusterComponentManager.GetByComponent(m.GetId())
	if err != nil {
		return err
	}
	for _, c := range cs {
		if err := c.Detach(ctx, userCred); err != nil {
			return err
		}
	}
	return m.Delete(ctx, userCred)
}

func (m *SComponent) DoDisable(cluster *SCluster) error {
	if err := m.SetEnabled(false); err != nil {
		return err
	}
	drv, err := m.GetDriver()
	if err != nil {
		return err
	}
	settings, err := m.GetSettings()
	if err != nil {
		return err
	}
	return drv.DoDisable(cluster, settings)
}

func (m *SComponent) FetchStatus(cluster *SCluster, out *apis.ComponentsStatus) error {
	drv, err := m.GetDriver()
	if err != nil {
		return err
	}
	return drv.FetchStatus(cluster, m, out)
}

func (m *SComponent) GetSettings() (*apis.ComponentSettings, error) {
	out := new(apis.ComponentSettings)
	if err := m.Settings.Unmarshal(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (m *SComponent) DoUpdate(cluster *SCluster, input *apis.ComponentUpdateInput) error {
	drv, err := m.GetDriver()
	if err != nil {
		return err
	}
	oldSettings, err := m.GetSettings()
	if err != nil {
		return err
	}
	settings, err := drv.GetUpdateSettings(oldSettings, input)
	if err != nil {
		return err
	}
	if _, err := db.Update(m, func() error {
		m.Settings = jsonutils.Marshal(settings)
		return nil
	}); err != nil {
		return err
	}
	if m.Enabled.Bool() {
		if err := drv.DoDisable(cluster, settings); err != nil {
			return err
		}
	}
	return drv.DoEnable(cluster, settings)
}
