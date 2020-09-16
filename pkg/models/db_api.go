package models

import (
	"context"
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	fedDBAPI IFedDBAPI
)

func init() {
	GetFedDBAPI()
}

func GetFedDBAPI() IFedDBAPI {
	if fedDBAPI == nil {
		fedDBAPI = newFedDBAPI()
	}
	return fedDBAPI
}

type IFedDBAPI interface {
	ClusterScope() IFedClusterDBAPI
	NamespaceScope() IFedNamespaceDBAPI
	JointDBAPI() IFedJointDBAPI

	// IsAttach2Cluster check federated object is attach to specified cluster
	IsAttach2Cluster(obj IFedModel, clusterId string) (bool, error)
	// PerformAttachCluster sync federated template object to cluster
	PerformAttachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, error)
	// PerformSyncCluster sync resource to cluster
	PerformSyncCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error
	// PerformDetachCluster delete federated releated object inside cluster
	PerformDetachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error
}

type sFedDBAPI struct {
	clusterScope   IFedClusterDBAPI
	namespaceScope IFedNamespaceDBAPI
	jointDBAPI     IFedJointDBAPI
}

func newFedDBAPI() IFedDBAPI {
	a := new(sFedDBAPI)
	clusterScope := newFedClusterDBAPI()
	namespaceScope := newFedNamespaceDBAPI()
	jointDBAPI := newFedJointDBAPI()
	a.clusterScope = clusterScope
	a.namespaceScope = namespaceScope
	a.jointDBAPI = jointDBAPI
	return a
}

func (a sFedDBAPI) ClusterScope() IFedClusterDBAPI {
	return a.clusterScope
}

func (a sFedDBAPI) NamespaceScope() IFedNamespaceDBAPI {
	return a.namespaceScope
}

func (a sFedDBAPI) JointDBAPI() IFedJointDBAPI {
	return a.jointDBAPI
}

func (a sFedDBAPI) GetJointModel(obj IFedModel, clusterId string) (IFedJointClusterModel, error) {
	jMan := obj.GetJointModelManager()
	jObj, err := GetFederatedJointClusterModel(jMan, obj.GetId(), clusterId)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	return jObj, nil
}

func (a sFedDBAPI) IsAttach2Cluster(obj IFedModel, clusterId string) (bool, error) {
	jObj, err := a.GetJointModel(obj, clusterId)
	if err != nil {
		return false, err
	}
	return jObj != nil, nil
}

func (a sFedDBAPI) PerformSyncCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	jObj, data, err := obj.ValidateJointCluster(userCred, data)
	if err != nil {
		return err
	}
	return a.performSyncCluster(jObj, ctx, userCred)
}

func (a sFedDBAPI) performSyncCluster(jObj IFedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	return a.jointDBAPI.ReconcileResource(jObj, ctx, userCred)
}

func (a sFedDBAPI) PerformAttachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) (IFedJointClusterModel, error) {
	data, err := obj.ValidateAttachCluster(ctx, userCred, data)
	if err != nil {
		return nil, err
	}
	clusterId, err := data.GetString("cluster_id")
	if err != nil {
		return nil, err
	}
	jObj, err := a.attachCluster(obj, ctx, userCred, clusterId)
	if err != nil {
		return nil, err
	}
	if err := a.performSyncCluster(jObj, ctx, userCred); err != nil {
		return nil, err
	}
	return jObj, err
}

func (a sFedDBAPI) attachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) (IFedJointClusterModel, error) {
	defer lockman.ReleaseObject(ctx, obj)
	lockman.LockObject(ctx, obj)

	cls, err := GetClusterManager().GetCluster(clusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s", clusterId)
	}

	attached, err := a.IsAttach2Cluster(obj, clusterId)
	if err != nil {
		return nil, errors.Wrap(err, "check IsAttach2Cluster")
	}
	if attached {
		return nil, errors.Errorf("%s %s has been attached to cluster %s", obj.Keyword(), obj.GetId(), clusterId)
	}
	jointMan := obj.GetJointModelManager()
	jointModel, err := db.NewModelObject(jointMan)
	if err != nil {
		return nil, errors.Wrapf(err, "new joint model %s", jointMan.Keyword())
	}
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(obj.GetId()), jointMan.GetMasterFieldName())
	data.Add(jsonutils.NewString(clusterId), jointMan.GetSlaveFieldName())
	if err := data.Unmarshal(jointModel); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if err := jointMan.TableSpec().Insert(ctx, jointModel); err != nil {
		return nil, errors.Wrap(err, "insert joint model")
	}
	db.OpsLog.LogAttachEvent(ctx, obj, cls, userCred, nil)
	return jointModel.(IFedJointClusterModel), nil
}

func (a sFedDBAPI) PerformDetachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	data, err := obj.ValidateDetachCluster(ctx, userCred, data)
	if err != nil {
		return err
	}
	clusterId, _ := data.GetString("cluster_id")
	return a.detachCluster(obj, ctx, userCred, clusterId)
}

func (a sFedDBAPI) detachCluster(obj IFedModel, ctx context.Context, userCred mcclient.TokenCredential, clusterId string) error {
	defer lockman.ReleaseObject(ctx, obj)
	lockman.LockObject(ctx, obj)

	attached, err := a.IsAttach2Cluster(obj, clusterId)
	if err != nil {
		return errors.Wrap(err, "check IsAttach2Cluster")
	}
	if !attached {
		return nil
	}

	jointModel, err := a.GetJointModel(obj, clusterId)
	if err != nil {
		return errors.Wrap(err, "detach get joint model")
	}

	// TODO: start task todo it
	return jointModel.Detach(ctx, userCred)
}

type IFedClusterDBAPI interface {
}

type sFedClusterDBAPI struct{}

func newFedClusterDBAPI() IFedClusterDBAPI {
	return &sFedClusterDBAPI{}
}

type IFedNamespaceDBAPI interface{}

type sFedNamespaceDBAPI struct{}

func newFedNamespaceDBAPI() IFedNamespaceDBAPI {
	return &sFedNamespaceDBAPI{}
}

type IFedJointDBAPI interface {
	ClusterScope() IFedJointClusterDBAPI
	NamespaceScope() IFedNamespaceJointClusterDBAPI

	// IsResourceExist check joint federated object's resource whether exists in target cluster
	IsResourceExist(jObj IFedJointClusterModel, userCred mcclient.TokenCredential) (IClusterModel, bool, error)
	// ReconcileResource reconcile federated object to cluster
	ReconcileResource(jObj IFedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error
	// FetchResourceModel get joint object releated cluster object
	FetchResourceModel(jObj IFedJointClusterModel) (IClusterModel, error)
	// FetchFedResourceModel get joint object related master fedreated db object
	FetchFedResourceModel(jObj IFedJointClusterModel) (IFedModel, error)
	// IsNamespaceScope mark object is namespace scope
	IsNamespaceScope(jObj IFedJointClusterModel) bool
	// GetDetails get joint object details
	GetDetails(jObj IFedJointClusterModel, userCred mcclient.TokenCredential, base apis.JointResourceBaseDetails, isList bool) interface{}
}

type sFedJointDBAPI struct {
	clusterScope   IFedJointClusterDBAPI
	namespaceScope IFedNamespaceJointClusterDBAPI
}

func newFedJointDBAPI() IFedJointDBAPI {
	clsScope := newFedJointClusterDBAPI()
	a := new(sFedJointDBAPI)
	a.clusterScope = clsScope
	a.namespaceScope = newFedNamespaceJointClusterDBAPI(a)
	return a
}

func (a sFedJointDBAPI) ClusterScope() IFedJointClusterDBAPI {
	return a.clusterScope
}

func (a sFedJointDBAPI) NamespaceScope() IFedNamespaceJointClusterDBAPI {
	return a.namespaceScope
}

func (a sFedJointDBAPI) IsNamespaceScope(jObj IFedJointClusterModel) bool {
	return jObj.GetResourceManager().IsNamespaceScope()
}

func (a sFedJointDBAPI) FetchResourceModel(jObj IFedJointClusterModel) (IClusterModel, error) {
	man := jObj.GetResourceManager()
	// namespace scope resource should also fetched by resourceId
	return FetchClusterResourceById(man, jObj.GetClusterId(), "", jObj.GetResourceId())
}

func (a sFedJointDBAPI) GetDetails(jObj IFedJointClusterModel, userCred mcclient.TokenCredential, base apis.JointResourceBaseDetails, isList bool) interface{} {
	out := api.FedJointClusterResourceDetails{
		JointResourceBaseDetails: base,
	}
	cluster, err := jObj.GetCluster()
	if err != nil {
		log.Errorf("get cluster %s object error: %v", jObj.GetClusterId(), err)
	} else {
		out.Cluster = cluster.GetName()
	}

	if fedObj, err := a.FetchFedResourceModel(jObj); err != nil {
		log.Errorf("get federated resource %s object error: %v", jObj.GetFedResourceId(), err)
	} else {
		out.FederatedResource = fedObj.GetName()
		out.FederatedResourceKeyword = fedObj.Keyword()
	}
	if a.IsNamespaceScope(jObj) {
		nsObj, err := a.namespaceScope.FetchClusterNamespace(userCred, jObj, cluster)
		if err == nil && nsObj != nil {
			out.Namespace = nsObj.GetName()
			out.NamespaceId = nsObj.GetId()
		}
	}
	if jObj.GetResourceId() != "" {
		resObj, err := a.FetchResourceModel(jObj)
		if err == nil && resObj != nil {
			out.Resource = resObj.GetName()
			out.ResourceStatus = resObj.GetStatus()
			out.ResourceKeyword = resObj.Keyword()
		}
	}
	return out
}

func (a sFedJointDBAPI) FetchResourceModelByName(jObj IFedJointClusterModel, userCred mcclient.TokenCredential) (IClusterModel, error) {
	cluster, err := jObj.GetCluster()
	if err != nil {
		return nil, errors.Wrapf(err, "get %s joint object cluster", jObj.Keyword())
	}
	fedObj, err := a.FetchFedResourceModel(jObj)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s related federated resource", jObj.Keyword())
	}
	clsNsId := ""
	checkResName := fedObj.GetName()
	if a.IsNamespaceScope(jObj) {
		nsObj, err := a.namespaceScope.FetchClusterNamespace(userCred, jObj, cluster)
		if err != nil {
			return nil, errors.Wrapf(err, "get %s cluster %s namespace", cluster.GetName(), jObj.Keyword())
		}
		clsNsId = nsObj.GetId()
	}
	man := jObj.GetResourceManager()
	resObj, err := FetchClusterResourceByIdOrName(man, userCred, cluster.GetId(), clsNsId, checkResName)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrapf(err, "get %s cluster %s resource %s", jObj.Keyword(), cluster.GetName(), checkResName)
	}
	if resObj != nil {
		// cluster resource object already exists
		return resObj, nil
	}
	return nil, nil
}

func (a sFedJointDBAPI) IsResourceExist(jObj IFedJointClusterModel, userCred mcclient.TokenCredential) (IClusterModel, bool, error) {
	if jObj.GetResourceId() == "" {
		// check cluster related same name resource whether exists
		resObj, err := a.FetchResourceModelByName(jObj, userCred)
		if err != nil {
			return nil, false, errors.Wrapf(err, "fetch joint %s resource model by name", jObj.Keyword())
		}
		if resObj != nil {
			// cluster resource object already exists
			return resObj, true, nil
		}
		return nil, false, nil
	}
	resObj, err := a.FetchResourceModel(jObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, errors.Wrapf(err, "FetchResourceModel of %s", jObj.Keyword())
	}
	if resObj == nil {
		return nil, false, nil
	}
	return resObj, true, nil
}

func (a sFedJointDBAPI) ReconcileResource(jObj IFedJointClusterModel, ctx context.Context, userCred mcclient.TokenCredential) error {
	resObj, exist, err := a.IsResourceExist(jObj, userCred)
	if err != nil {
		return errors.Wrapf(err, "Check %s/%s cluster resource %s %s exist", jObj.Keyword(), jObj.GetName(), resObj.Keyword(), resObj.GetName())
	}
	cluster, err := jObj.GetCluster()
	if err != nil {
		return errors.Wrap(err, "get joint object cluster")
	}
	ownerId := cluster.GetOwnerId()
	fedObj := db.JointMaster(jObj).(IFedModel)
	if exist {
		if err := a.updateResource(ctx, userCred, jObj, resObj); err != nil {
			return errors.Wrap(err, "UpdateClusterResource")
		}
		return nil
	}
	if err := a.createResource(ctx, userCred, ownerId, jObj, fedObj, cluster); err != nil {
		return errors.Wrap(err, "CreateClusterResource")
	}
	return nil
}

func (a sFedJointDBAPI) createResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	jObj IFedJointClusterModel,
	fedObj IFedModel,
	cluster *SCluster,
) error {
	baseInput := new(api.NamespaceResourceCreateInput)
	baseInput.Name = fedObj.GetName()
	baseInput.ClusterId = cluster.GetId()
	baseInput.ProjectDomainId = cluster.DomainId
	if a.IsNamespaceScope(jObj) {
		clsNs, err := a.namespaceScope.FetchClusterNamespace(userCred, jObj, cluster)
		if err != nil {
			return errors.Wrapf(err, "get %s cluster %s namespace", jObj.Keyword(), cluster.GetName())
		}
		baseInput.NamespaceId = clsNs.GetId()
	}
	data, err := jObj.GetResourceCreateData(ctx, userCred, *baseInput)
	if err != nil {
		return errors.Wrapf(err, "get fed joint resource %s create to cluster %s resource data", jObj.Keyword(), cluster.GetName())
	}
	resObj, err := db.DoCreate(jObj.GetResourceManager(), ctx, userCred, nil, data, ownerId)
	if err != nil {
		return errors.Wrapf(err, "create cluster %s %s local resource object", cluster.GetName(), jObj.GetResourceManager().Keyword())
	}
	if err := jObj.SetResource(resObj.(IClusterModel)); err != nil {
		return errors.Wrapf(err, "set %s resource object", jObj.Keyword())
	}
	func() {
		lockman.LockObject(ctx, resObj)
		defer lockman.ReleaseObject(ctx, resObj)

		resObj.PostCreate(ctx, userCred, ownerId, nil, data)
	}()
	return nil
}

func (a sFedJointDBAPI) updateResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	jObj IFedJointClusterModel,
	resObj IClusterModel,
) error {
	if err := jObj.SetResource(resObj.(IClusterModel)); err != nil {
		return errors.Wrapf(err, "set %s resource object", jObj.Keyword())
	}
	resObj.PostUpdate(ctx, userCred, nil, jsonutils.NewDict())
	return nil
}

func (_ sFedJointDBAPI) FetchFedResourceModel(jObj IFedJointClusterModel) (IFedModel, error) {
	fedMan := jObj.GetManager().GetFedManager()
	fObj, err := fedMan.FetchById(jObj.GetFedResourceId())
	if err != nil {
		return nil, errors.Wrapf(err, "get federated resource %s by id %s", fedMan.Keyword(), jObj.GetFedResourceId())
	}
	return fObj.(IFedModel), nil
}

type IFedJointClusterDBAPI interface {
}

type fedJointClusterDBAPI struct{}

func newFedJointClusterDBAPI() IFedJointClusterDBAPI {
	return &fedJointClusterDBAPI{}
}

type IFedNamespaceJointClusterDBAPI interface {
	FetchFedNamespace(jObj IFedNamespaceJointClusterModel) (*SFedNamespace, error)
	FetchClusterNamespace(userCred mcclient.TokenCredential, jObj IFedNamespaceJointClusterModel, cluster *SCluster) (*SNamespace, error)
	FetchModelsByFednamespace(man IFedNamespaceJointClusterManager, fednsId string) ([]IFedNamespaceJointClusterModel, error)
}

type fedNamespaceJointClusterDBAPI struct {
	jointDBAPI IFedJointDBAPI
}

func newFedNamespaceJointClusterDBAPI(
	jointDBAPI IFedJointDBAPI,
) IFedNamespaceJointClusterDBAPI {
	return &fedNamespaceJointClusterDBAPI{
		jointDBAPI: jointDBAPI,
	}
}

func (a fedNamespaceJointClusterDBAPI) FetchFedNamespace(jObj IFedNamespaceJointClusterModel) (*SFedNamespace, error) {
	fedObj, err := a.jointDBAPI.FetchFedResourceModel(jObj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get fed %s obj", jObj.GetModelManager().Keyword())
	}
	return fedObj.(IFedNamespaceModel).GetFedNamespace()
}

func (a fedNamespaceJointClusterDBAPI) FetchClusterNamespace(userCred mcclient.TokenCredential, jObj IFedNamespaceJointClusterModel, cluster *SCluster) (*SNamespace, error) {
	fedNs, err := a.FetchFedNamespace(jObj)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s federatednamespace", jObj.Keyword())
	}
	nsName := fedNs.GetName()
	nsObj, err := GetNamespaceManager().GetByIdOrName(userCred, cluster.GetId(), nsName)
	if err != nil {
		return nil, errors.Wrapf(err, "get cluster %s namespace %s", cluster.GetName(), nsName)
	}
	return nsObj.(*SNamespace), nil
}

func (_ fedNamespaceJointClusterDBAPI) FetchModelsByFednamespace(m IFedNamespaceJointClusterManager, fednsId string) ([]IFedNamespaceJointClusterModel, error) {
	objs := make([]interface{}, 0)
	sq := m.GetFedManager().Query("id").Equals("federatednamespace_id", fednsId).SubQuery()
	q := m.Query().In("federatedresource_id", sq)
	if err := db.FetchModelObjects(m, q, &objs); err != nil {
		return nil, err
	}
	ret := make([]IFedNamespaceJointClusterModel, len(objs))
	for i := range objs {
		obj := objs[i]
		objPtr := GetObjectPtr(obj).(IFedNamespaceJointClusterModel)
		ret[i] = objPtr
	}
	return ret, nil
}
