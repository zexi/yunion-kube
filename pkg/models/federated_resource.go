package models

import (
	"context"
	"database/sql"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

type IFederatedModelManager interface {
	db.IModelManager

	GetJointModelManager() IFederatedJointClusterManager
	SetJointModelManager(man IFederatedJointClusterManager)

	PerformAttachCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFederatedJointClusterModel, error)
	PerformDetachCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error
	PerformSyncCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error
}

type IFederatedModel interface {
	db.IModel

	GetManager() IFederatedModelManager
	GetDetails(baseDetails interface{}, isList bool) interface{}
	ValidateJointCluster(userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFederatedJointClusterModel, jsonutils.JSONObject, error)
	GetJointModelManager() IFederatedJointClusterManager
	ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	ValidateDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

// +onecloud:swagger-gen-ignore
type SFederatedResourceBaseManager struct {
	db.SStatusDomainLevelResourceBaseManager
	jointManager IFederatedJointClusterManager
}

type SFederatedResourceBase struct {
	db.SStatusDomainLevelResourceBase
}

func NewFedResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SFederatedResourceBaseManager {
	return SFederatedResourceBaseManager{
		SStatusDomainLevelResourceBaseManager: db.NewStatusDomainLevelResourceBaseManager(
			dt, tableName, keyword, keywordPlural),
	}
}

func (m *SFederatedResourceBaseManager) SetJointModelManager(man IFederatedJointClusterManager) {
	m.jointManager = man
}

func (m *SFederatedResourceBaseManager) GetJointModelManager() IFederatedJointClusterManager {
	return m.jointManager
}

func (m *SFederatedResourceBase) GetJointModelManager() IFederatedJointClusterManager {
	return m.GetManager().GetJointModelManager()
}

func (m *SFederatedResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusDomainLevelResourceListInput)
}

func (m *SFederatedResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedResourceCreateInput) (*api.FederatedResourceCreateInput, error) {
	dInput, err := m.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceCreateInput = dInput
	return input, nil
}

func (m *SFederatedResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []interface{} {
	baseGet := func(obj interface{}) interface{} {
		vRows := m.SStatusDomainLevelResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{obj}, fields, isList)
		return vRows[0]
	}
	ret := make([]interface{}, len(objs))
	for idx := range objs {
		obj := objs[idx].(IFederatedModel)
		baseDetail := baseGet(obj)
		out := obj.GetDetails(baseDetail, isList)
		ret[idx] = out
	}
	return ret
}

func (obj *SFederatedResourceBase) GetManager() IFederatedModelManager {
	return obj.GetModelManager().(IFederatedModelManager)
}

func (obj *SFederatedResourceBase) GetClustersQuery() *sqlchemy.SQuery {
	jointMan := obj.GetJointModelManager()
	return jointMan.Query().Equals(jointMan.GetMasterFieldName(), obj.GetId())
}

func (obj *SFederatedResourceBase) GetClustersCount() (int, error) {
	q := obj.GetClustersQuery()
	return q.CountWithError()
}

func (obj *SFederatedResourceBase) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedResourceDetails{
		StatusDomainLevelResourceDetails: base.(apis.StatusDomainLevelResourceDetails),
	}
	if isList {
		return out
	}
	// placement := api.FederatedPlacement{}
	return out
}

func (obj *SFederatedResourceBase) AllowPerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, obj, "attach-cluster")
}

func (obj *SFederatedResourceBase) ValidateJointCluster(userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFederatedJointClusterModel, jsonutils.JSONObject, error) {
	jointMan := obj.GetJointModelManager()
	clusterId, _ := data.GetString("cluster_id")
	if clusterId == "" {
		return nil, data, httperrors.NewInputParameterError("cluster_id not provided")
	}
	cluster, err := GetClusterManager().GetClusterByIdOrName(userCred, clusterId)
	if err != nil {
		return nil, nil, err
	}
	clusterId = cluster.GetId()
	data.(*jsonutils.JSONDict).Set("cluster_id", jsonutils.NewString(clusterId))
	jointModel, err := GetFederatedJointClusterModel(jointMan, obj.GetId(), clusterId)
	return jointModel, data, err
}

func (obj *SFederatedResourceBase) ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jointModel, data, err := obj.ValidateJointCluster(userCred, data)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return data, err
	}
	clusterId, _ := data.GetString("cluster_id")
	if jointModel != nil {
		return data, httperrors.NewInputParameterError("cluster %s has been attached", clusterId)
	}
	return data, nil
}

func (m *SFederatedResourceBaseManager) GetJointModel(model IFederatedModel, clusterId string) (IFederatedJointClusterModel, error) {
	jointMan := model.GetJointModelManager()
	jointModel, err := GetFederatedJointClusterModel(jointMan, model.GetId(), clusterId)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	return jointModel, nil
}

func (m *SFederatedResourceBaseManager) IsAttach2Cluster(model IFederatedModel, clusterId string) (bool, error) {
	jointModel, err := m.GetJointModel(model, clusterId)
	if err != nil {
		return false, err
	}
	return jointModel != nil, nil
}

func (m *SFederatedResourceBaseManager) PerformAttachCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFederatedJointClusterModel, error) {
	data, err := model.ValidateAttachCluster(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	clusterId, err := data.GetString("cluster_id")
	if err != nil {
		return nil, err
	}
	jointObj, err := m.attachCluster(model, ctx, userCred, clusterId)
	if err != nil {
		return nil, err
	}
	if err := m.performSyncCluster(jointObj, ctx, userCred); err != nil {
		return nil, err
	}
	return jointObj, nil
}

func (m *SFederatedResourceBaseManager) attachCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) (IFederatedJointClusterModel, error) {
	defer lockman.ReleaseObject(ctx, model)
	lockman.LockObject(ctx, model)

	cls, err := GetClusterManager().GetCluster(clusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s", clusterId)
	}

	attached, err := m.IsAttach2Cluster(model, clusterId)
	if err != nil {
		return nil, errors.Wrap(err, "check IsAttach2Cluster")
	}
	if attached {
		return nil, errors.Errorf("%s %s has been attached to cluster %s", model.Keyword(), model.GetId(), clusterId)
	}
	jointMan := model.GetJointModelManager()
	jointModel, err := db.NewModelObject(jointMan)
	if err != nil {
		return nil, errors.Wrapf(err, "new joint model %s", jointMan.Keyword())
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(model.GetId()), jointMan.GetMasterFieldName())
	data.Add(jsonutils.NewString(clusterId), jointMan.GetSlaveFieldName())
	if err := data.Unmarshal(jointModel); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if err := jointMan.TableSpec().Insert(ctx, jointModel); err != nil {
		return nil, errors.Wrap(err, "insert joint model")
	}
	db.OpsLog.LogAttachEvent(ctx, model, cls, userCred, nil)
	return jointModel.(IFederatedJointClusterModel), nil
}

func (obj *SFederatedResourceBase) GetK8sObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: obj.Name,
	}
}

func (obj *SFederatedResourceBase) AllowPerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return db.IsDomainAllowPerform(userCred, obj, "detach-cluster")
}

func (obj *SFederatedResourceBase) ValidateDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	jointModel, input, err := obj.ValidateJointCluster(userCred, data)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return input, err
	}
	clusterId, _ := data.GetString("cluster_id")
	if jointModel == nil {
		return input, httperrors.NewInputParameterError("cluster %s has not been attached", clusterId)
	}
	return input, nil
}

func (m *SFederatedResourceBaseManager) PerformDetachCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	data, err := model.ValidateDetachCluster(ctx, userCred, data)
	if err != nil {
		return err
	}
	clusterId, _ := data.GetString("cluster_id")
	return m.detachCluster(model, ctx, userCred, clusterId)
}

func (m *SFederatedResourceBaseManager) detachCluster(model IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) error {
	defer lockman.ReleaseObject(ctx, model)
	lockman.LockObject(ctx, model)

	attached, err := m.IsAttach2Cluster(model, clusterId)
	if err != nil {
		return errors.Wrap(err, "check IsAttach2Cluster")
	}
	if !attached {
		return nil
	}

	jointModel, err := m.GetJointModel(model, clusterId)
	if err != nil {
		return errors.Wrap(err, "detach get joint model")
	}

	// TODO: start task todo it
	return jointModel.Detach(ctx, userCred)
}

func (m *SFederatedResourceBaseManager) PerformSyncCluster(obj IFederatedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	jointObj, data, err := obj.ValidateJointCluster(userCred, data)
	if err != nil {
		return err
	}
	return m.performSyncCluster(jointObj, ctx, userCred)
}

func (m *SFederatedResourceBaseManager) performSyncCluster(jointObj IFederatedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := jointObj.GetManager().PerformSyncResource(jointObj, ctx, userCred); err != nil {
		return errors.Wrapf(err, "PerformSyncResource for %s", jointObj.Keyword())
	}
	return nil
}

func (m *SFederatedResourceBase) PerformSyncCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	return nil, m.GetManager().PerformSyncCluster(m, ctx, userCred, data.JSON(data))
}

func (m *SFederatedResourceBase) GetAttachedClusters(ctx context.Context) ([]SCluster, error) {
	jm := m.GetJointModelManager()
	clusters := make([]SCluster, 0)
	q := GetClusterManager().Query()
	sq := jm.Query("cluster_id").Equals("federatedresource_id", m.GetId()).SubQuery()
	q = q.In("id", sq)
	if err := db.FetchModelObjects(GetClusterManager(), q, &clusters); err != nil {
		return nil, errors.Wrapf(err, "get federated resource %s %s attached clusters", m.Keyword(), m.GetName())
	}
	return clusters, nil
}

func (m *SFederatedResourceBase) ValidateDeleteCondition(ctx context.Context) error {
	clusters, err := m.GetAttachedClusters(ctx)
	if err != nil {
		return errors.Wrap(err, "get attached clusters")
	}
	if len(clusters) != 0 {
		return httperrors.NewNotEmptyError("federated resource %s attached to %d clusters", m.Keyword(), len(clusters))
	}
	return nil
}
