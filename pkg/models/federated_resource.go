package models

import (
	"context"
	"database/sql"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

type IFedModelManager interface {
	db.IModelManager

	GetJointModelManager() IFedJointClusterManager
	SetJointModelManager(man IFedJointClusterManager)
}

type IFedModel interface {
	db.IModel

	GetManager() IFedModelManager
	GetDetails(baseDetails interface{}, isList bool) interface{}
	ValidateJointCluster(userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, jsonutils.JSONObject, error)
	GetJointModelManager() IFedJointClusterManager
	ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
	ValidateDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error)
}

// +onecloud:swagger-gen-ignore
type SFedResourceBaseManager struct {
	db.SStatusDomainLevelResourceBaseManager
	jointManager IFedJointClusterManager
}

type SFedResourceBase struct {
	db.SStatusDomainLevelResourceBase
}

func NewFedResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SFedResourceBaseManager {
	return SFedResourceBaseManager{
		SStatusDomainLevelResourceBaseManager: db.NewStatusDomainLevelResourceBaseManager(
			dt, tableName, keyword, keywordPlural),
	}
}

func (m *SFedResourceBaseManager) SetJointModelManager(man IFedJointClusterManager) {
	m.jointManager = man
}

func (m *SFedResourceBaseManager) GetJointModelManager() IFedJointClusterManager {
	return m.jointManager
}

func (m *SFedResourceBase) GetJointModelManager() IFedJointClusterManager {
	return m.GetManager().GetJointModelManager()
}

func (m *SFedResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SStatusDomainLevelResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StatusDomainLevelResourceListInput)
}

func (m *SFedResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedResourceCreateInput) (*api.FederatedResourceCreateInput, error) {
	dInput, err := m.SStatusDomainLevelResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, input.StatusDomainLevelResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceCreateInput = dInput
	return input, nil
}

func (m *SFedResourceBaseManager) FetchCustomizeColumns(
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
		obj := objs[idx].(IFedModel)
		baseDetail := baseGet(obj)
		out := obj.GetDetails(baseDetail, isList)
		ret[idx] = out
	}
	return ret
}

func (obj *SFedResourceBase) GetManager() IFedModelManager {
	return obj.GetModelManager().(IFedModelManager)
}

func (obj *SFedResourceBase) GetClustersQuery() *sqlchemy.SQuery {
	jointMan := obj.GetJointModelManager()
	return jointMan.Query().Equals(jointMan.GetMasterFieldName(), obj.GetId())
}

func (obj *SFedResourceBase) GetClustersCount() (int, error) {
	q := obj.GetClustersQuery()
	return q.CountWithError()
}

func (obj *SFedResourceBase) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedResourceDetails{
		StatusDomainLevelResourceDetails: base.(apis.StatusDomainLevelResourceDetails),
	}
	if isList {
		return out
	}
	// placement := api.FederatedPlacement{}
	return out
}

func (obj *SFedResourceBase) ValidateJointCluster(userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, jsonutils.JSONObject, error) {
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

func (obj *SFedResourceBase) ValidateAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

func (obj *SFedResourceBase) GetK8sObjectMeta() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name: obj.Name,
	}
}

func (obj *SFedResourceBase) ValidateDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
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

func (obj *SFedResourceBase) GetElemModel() (IFedModel, error) {
	m := obj.GetManager()
	elemObj, err := db.FetchById(m, obj.GetId())
	if err != nil {
		return nil, err
	}
	return elemObj.(IFedModel), nil
}

func (obj *SFedResourceBase) PerformSyncCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedResourceJointClusterInput) (*api.FederatedResourceJointClusterInput, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	return nil, GetFedResAPI().PerformSyncCluster(elemObj, ctx, userCred, data.JSON(data))
}

func (obj *SFedResourceBase) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	if _, err := GetFedResAPI().PerformAttachCluster(elemObj, ctx, userCred, data); err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFedResourceBase) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	elemObj, err := obj.GetElemModel()
	if err != nil {
		return nil, err
	}
	return nil, GetFedResAPI().PerformDetachCluster(elemObj, ctx, userCred, data)
}

// TODO: move to api interface
func (m *SFedResourceBase) GetAttachedClusters(ctx context.Context) ([]SCluster, error) {
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

func (m *SFedResourceBase) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.FedResourceUpdateInput) (*api.FedResourceUpdateInput, error) {
	bInput, err := m.SStatusDomainLevelResourceBase.ValidateUpdateData(ctx, userCred, query, input.StatusDomainLevelResourceBaseUpdateInput)
	if err != nil {
		return nil, err
	}
	input.StatusDomainLevelResourceBaseUpdateInput = bInput
	if input.Name != "" {
		return nil, httperrors.NewInputParameterError("Can not update name")
	}
	return input, nil
}

func (m *SFedResourceBase) ValidateDeleteCondition(ctx context.Context) error {
	clusters, err := m.GetAttachedClusters(ctx)
	if err != nil {
		return errors.Wrap(err, "get attached clusters")
	}
	clsName := make([]string, len(clusters))
	for i := range clusters {
		clsName[i] = clusters[i].GetName()
	}
	if len(clusters) != 0 {
		return httperrors.NewNotEmptyError("federated resource %s attached to cluster %v", m.Keyword(), clsName)
	}
	return nil
}
