package models

import (
	"context"
	"database/sql"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

type SFederatedJointResourceBaseManager struct {
	db.SJointResourceBaseManager
}

type SFederatedJointResourceBase struct {
	db.SJointResourceBase
}

type IFederatedJointModel interface {
	db.IJointModel

	GetDetails(baseDetails interface{}, isList bool) interface{}
}

type IFederatedJointManager interface {
	db.IJointModelManager
}

type IFederatedJointClusterManager interface {
	IFederatedJointManager
	GetResourceManager() IClusterModelManager
	ClusterQuery(clusterId string) *sqlchemy.SQuery

	PerformSyncResource(jointObj IFederatedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error
}

type IFederatedJointClusterModel interface {
	IFederatedJointModel

	GetManager() IFederatedJointClusterManager
	GetCluster() (*SCluster, error)
	GetResourceManager() IClusterModelManager
	IsNamespaceScope() bool
	IsResourceExist() (IClusterModel, bool, error)
	SetResource(resObj IClusterModel) error
	GetResourceCreateData(baseInput api.ClusterResourceCreateInput) (jsonutils.JSONObject, error)
	UpdateResource(resObj IClusterModel) error
}

func NewFederatedJointResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string, master, slave db.IStandaloneModelManager) SFederatedJointResourceBaseManager {
	return SFederatedJointResourceBaseManager{
		SJointResourceBaseManager: db.NewJointResourceBaseManager(dt, tableName, keyword, keywordPlural, master, slave),
	}
}

func NewFederatedJointClusterManager(
	dt interface{}, tableName string,
	keyword string, keywordPlural string,
	master db.IStandaloneModelManager,
	resourceMan IClusterModelManager,
) SFederatedJointClusterManager {
	base := NewFederatedJointResourceBaseManager(dt, tableName, keyword, keywordPlural, master, GetClusterManager())
	return SFederatedJointClusterManager{
		SFederatedJointResourceBaseManager: base,
		masterFieldName:                    fmt.Sprintf("%s_id", master.Keyword()),
		resourceManager:                    resourceMan,
	}
}

func NewFederatedJointManager(factory func() db.IJointModelManager) db.IJointModelManager {
	man := factory()
	man.SetVirtualObject(man)
	return man
}

type SFederatedJointClusterManager struct {
	SFederatedJointResourceBaseManager
	masterFieldName string
	resourceManager IClusterModelManager
}

type SFederatedJointCluster struct {
	SFederatedJointResourceBase

	ClusterId   string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	NamespaceId string `width:"36" charset:"ascii" list:"user" index:"true"`
	ResourceId  string `width:"36" charset:"ascii" list:"user" index:"true"`
}

func (m SFederatedJointClusterManager) GetResourceManager() IClusterModelManager {
	return m.resourceManager
}

func (m SFederatedJointClusterManager) GetMasterFieldName() string {
	return m.masterFieldName
}

func (m SFederatedJointClusterManager) GetSlaveFieldName() string {
	return "cluster_id"
}

func (m *SFederatedJointClusterManager) ClusterQuery(clsId string) *sqlchemy.SQuery {
	return m.Query().Equals("cluster_id", clsId)
}

func GetFederatedJointClusterModel(man IFederatedJointClusterManager, masterId string, clusterId string) (IFederatedJointClusterModel, error) {
	q := man.ClusterQuery(clusterId).Equals(man.GetMasterFieldName(), masterId)
	obj, err := db.NewModelObject(man)
	if err != nil {
		return nil, errors.Wrapf(err, "NewModelObject %s", man.Keyword())
	}
	if err := q.First(obj); err != nil {
		return nil, err
	}
	return obj.(IFederatedJointClusterModel), nil
}

func (obj *SFederatedJointCluster) GetManager() IFederatedJointClusterManager {
	return obj.GetJointModelManager().(IFederatedJointClusterManager)
}

func (obj *SFederatedJointCluster) GetResourceManager() IClusterModelManager {
	return obj.GetManager().GetResourceManager()
}

func (obj *SFederatedJointCluster) IsNamespaceScope() bool {
	return obj.GetResourceManager().IsNamespaceScope()
}

func (obj *SFederatedJointCluster) SetResource(resObj IClusterModel) error {
	_, err := db.Update(obj, func() error {
		obj.ResourceId = resObj.GetId()
		if obj.IsNamespaceScope() {
			obj.NamespaceId = resObj.(INamespaceModel).GetNamespaceId()
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "set resource_id")
	}
	return nil
}

func (obj *SFederatedJointCluster) UpdateResource(resObj IClusterModel) error {
	return nil
}

func (m *SFederatedJointClusterManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []interface{} {
	baseGet := func(obj interface{}) interface{} {
		jRows := m.SJointResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{obj}, fields, isList)
		return jRows[0]
	}
	ret := make([]interface{}, len(objs))
	for idx := range objs {
		obj := objs[idx].(IFederatedJointModel)
		baseDetail := baseGet(obj)
		out := obj.GetDetails(baseDetail, isList)
		ret[idx] = out
	}
	return ret
}

func (obj *SFederatedJointCluster) GetCluster() (*SCluster, error) {
	return GetClusterManager().GetCluster(obj.ClusterId)
}

func (obj *SFederatedJointCluster) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedJointClusterResourceDetails{
		JointResourceBaseDetails: base.(apis.JointResourceBaseDetails),
	}
	if cluster, err := obj.GetCluster(); err != nil {
		log.Errorf("get cluster %s object error: %v", obj.ClusterId, err)
	} else {
		out.Cluster = cluster.GetName()
	}
	if obj.IsNamespaceScope() {
		if obj.NamespaceId != "" {
			nsObj, err := GetNamespaceManager().FetchById(obj.NamespaceId)
			if err == nil && nsObj != nil {
				out.Namespace = nsObj.GetName()
			}
		}
	}
	if obj.ResourceId != "" {
		resObj, err := obj.GetResourceModel()
		if err == nil && resObj != nil {
			out.Resource = resObj.GetName()
		}
	}
	return out
}

func (obj *SFederatedJointCluster) GetResourceModel() (IClusterModel, error) {
	man := obj.GetResourceManager()
	return FetchClusterResourceById(man, obj.ClusterId, obj.NamespaceId, obj.ResourceId)
}

func (obj *SFederatedJointCluster) IsResourceExist() (IClusterModel, bool, error) {
	if obj.ResourceId == "" {
		return nil, false, nil
	}
	resObj, err := obj.GetResourceModel()
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrap(err, "GetResourceModel")
	}
	if resObj == nil {
		return nil, false, nil
	}
	return resObj, true, nil
}

func (m *SFederatedJointClusterManager) PerformSyncResource(
	jointObj IFederatedJointClusterModel,
	ctx context.Context,
	userCred mcclient.TokenCredential,
) error {
	return m.ReconcileResource(jointObj, ctx, userCred)
}

func (m *SFederatedJointClusterManager) ReconcileResource(
	jointObj IFederatedJointClusterModel,
	ctx context.Context,
	userCred mcclient.TokenCredential,
) error {
	resObj, exist, err := jointObj.IsResourceExist()
	if err != nil {
		return errors.Wrap(err, "Check cluster resource exist")
	}
	cluster, err := jointObj.GetCluster()
	if err != nil {
		return errors.Wrap(err, "get joint object cluster")
	}
	ownerId := cluster.GetOwnerId()
	fedObj := db.JointMaster(jointObj).(IFederatedModel)
	if exist {
		if err := m.UpdateResource(ctx, userCred, jointObj, resObj); err != nil {
			return errors.Wrap(err, "UpdateClusterResource")
		}
		return nil
	}
	if err := m.CreateResource(ctx, userCred, ownerId, jointObj, fedObj, cluster); err != nil {
		return errors.Wrap(err, "CreateClusterResource")
	}
	return nil
}

func (m *SFederatedJointClusterManager) CreateResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	jointObj IFederatedJointClusterModel,
	fedObj IFederatedModel,
	cluster *SCluster,
) error {
	baseInput := new(api.ClusterResourceCreateInput)
	baseInput.Name = fedObj.GetName()
	baseInput.ClusterId = cluster.GetId()
	baseInput.ProjectDomainId = cluster.DomainId
	data, err := jointObj.GetResourceCreateData(*baseInput)
	if err != nil {
		return errors.Wrap(err, "GetResourceCreateData")
	}
	resObj, err := db.DoCreate(jointObj.GetResourceManager(), ctx, userCred, nil, data, ownerId)
	if err != nil {
		return errors.Wrap(err, "create local resource object")
	}
	if err := jointObj.SetResource(resObj.(IClusterModel)); err != nil {
		return errors.Wrapf(err, "set %s resource object", jointObj.Keyword())
	}
	func() {
		lockman.LockObject(ctx, resObj)
		defer lockman.ReleaseObject(ctx, resObj)

		resObj.PostCreate(ctx, userCred, ownerId, nil, data)
	}()
	return nil
}

func (m *SFederatedJointClusterManager) UpdateResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	jointObj IFederatedJointClusterModel,
	resObj IClusterModel,
) error {
	diff, err := db.UpdateWithLock(ctx, resObj, func() error {
		return jointObj.UpdateResource(resObj)
	})
	if err != nil {
		return errors.Wrap(err, "UpdateResource")
	}
	db.OpsLog.LogEvent(resObj, db.ACT_UPDATE, diff, userCred)
	resObj.PostUpdate(ctx, userCred, nil, jsonutils.NewDict())
	return nil
}
