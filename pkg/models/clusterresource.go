package models

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

type SClusterResourceBaseManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager

	// resourceName is kubernetes resource name
	resourceName string
	// kindName is kubernetes resource kind
	kindName string
	// RawObject is kubernetes runtime object
	rawObject runtime.Object
}

func NewClusterResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
	resName string,
	kind string,
	object runtime.Object) SClusterResourceBaseManager {
	return SClusterResourceBaseManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt, tableName, keyword, keywordPlural),
		resourceName: resName,
		kindName:     kind,
		rawObject:    object,
	}
}

type SClusterResourceBase struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	ClusterId string `width:"36" charset:"ascii" nullable:"false" index:"true" list:"user"`
	// ResourceVersion is k8s remote object resourceVersion
	ResourceVersion string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

type IClusterModelManager interface {
	db.IVirtualModelManager

	GetK8SResourceInfo() K8SResourceInfo
	IsRemoteObjectLocalExist(userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, bool, error)
	NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error)
	ListRemoteObjects(cli *client.ClusterManager) ([]interface{}, error)
	GetRemoteObjectGlobalId(cluster *SCluster, obj interface{}) string

	NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error)
	CreateRemoteObject(model IClusterModel, cli *client.ClusterManager, remoteObj interface{}) (interface{}, error)
}

type IClusterModel interface {
	db.IVirtualModel

	GetExternalId() string
	SetExternalId(idStr string)

	SetName(name string)
	GetClusterId() string
	GetCluster() (*SCluster, error)
	SetCluster(userCred mcclient.TokenCredential, cluster *SCluster)
	SetStatus(userCred mcclient.TokenCredential, status string, reason string) error
	UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error
	GetClusterClient() (*client.ClusterManager, error)
	DeleteRemoteObject(cli *client.ClusterManager) error
	RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error
}

type K8SResourceInfo struct {
	ResourceName string
	KindName     string
	Object       runtime.Object
}

func (m SClusterResourceBaseManager) GetK8SResourceInfo() K8SResourceInfo {
	return K8SResourceInfo{
		ResourceName: m.resourceName,
		KindName:     m.kindName,
		Object:       m.rawObject,
	}
}

func (m *SClusterResourceBaseManager) FilterBySystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = m.SVirtualResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	input := new(api.ClusterResourceListInput)
	if input.Cluster != "" {
		clsObj, err := ClusterManager.FetchClusterByIdOrName(userCred, input.Cluster)
		if err != nil {
			log.Errorf("Get cluster %s error: %v", input.Cluster, err)
		}
		q.Equals("cluster_id", clsObj.GetId())
	}
	return q
}

func (m SClusterResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ClusterResourceCreateInput) (*api.ClusterResourceCreateInput, error) {
	vData, err := m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, data.VirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.VirtualResourceCreateInput = vData

	if data.Cluster == "" {
		return nil, httperrors.NewNotEmptyError("cluster is empty")
	}
	clsObj, err := ClusterManager.FetchByIdOrName(userCred, data.Cluster)
	if err != nil {
		return nil, NewCheckIdOrNameError("cluster", data.Cluster, err)
	}
	data.Cluster = clsObj.GetId()

	return data, nil
}

func FetchClusterResourceByName(manager IClusterModelManager, userCred mcclient.IIdentityProvider, clusterId string, namespaceId string, resId string) (IClusterModel, error) {
	if len(clusterId) == 0 {
		return nil, errors.Errorf("cluster id must provided")
	}
	q := manager.Query()
	q = manager.FilterByName(q, resId)
	q = q.Equals("cluster_id", clusterId)
	if namespaceId != "" {
		q = q.Equals("namespace_id", namespaceId)
	}
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count > 0 && userCred != nil {
		q = manager.FilterByOwner(q, userCred, manager.NamespaceScope())
		q = manager.FilterBySystemAttributes(q, nil, nil, manager.ResourceScope())
		count, err = q.CountWithError()
		if err != nil {
			return nil, err
		}
	}
	if count == 1 {
		obj, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		if err := q.First(obj); err != nil {
			return nil, err
		} else {
			return obj.(IClusterModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func FetchClusterResourceById(manager IClusterModelManager, clusterId string, namespaceId string, resId string) (IClusterModel, error) {
	if len(clusterId) == 0 {
		return nil, errors.Errorf("cluster id must provided")
	}
	q := manager.Query()
	q = manager.FilterById(q, resId)
	q = q.Equals("cluster_id", clusterId)
	if namespaceId != "" {
		q = q.Equals("namespace_id", namespaceId)
	}
	count, err := q.CountWithError()
	if err != nil {
		return nil, err
	}
	if count == 1 {
		obj, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		if err := q.First(obj); err != nil {
			return nil, err
		} else {
			return obj.(IClusterModel), nil
		}
	} else if count > 1 {
		return nil, sqlchemy.ErrDuplicateEntry
	} else {
		return nil, sql.ErrNoRows
	}
}

func FetchClusterResourceByIdOrName(manager IClusterModelManager, userCred mcclient.IIdentityProvider, clusterId string, namespaceId string, resId string) (IClusterModel, error) {
	if stringutils2.IsUtf8(resId) {
		return FetchClusterResourceByName(manager, userCred, clusterId, namespaceId, resId)
	}
	obj, err := FetchClusterResourceById(manager, clusterId, namespaceId, resId)
	if err == sql.ErrNoRows {
		return FetchClusterResourceByName(manager, userCred, clusterId, namespaceId, resId)
	} else {
		return obj, err
	}
}

func (m *SClusterResourceBaseManager) GetByIdOrName(userCred mcclient.IIdentityProvider, clusterId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByIdOrName(m, userCred, clusterId, "", resId)
}

func (m *SClusterResourceBaseManager) GetByName(userCred mcclient.IIdentityProvider, clusterId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByName(m, userCred, clusterId, "", resId)
}

func (res *SClusterResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := res.SVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.ClusterResourceCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "cluster resource unmarshal data")
	}
	res.ClusterId = input.Cluster
	return nil
}

func (res *SClusterResourceBase) GetParentId() string {
	return res.ClusterId
}

func (res *SClusterResourceBase) GetCluster() (*SCluster, error) {
	obj, err := ClusterManager.FetchById(res.ClusterId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch cluster %s", res.ClusterId)
	}
	return obj.(*SCluster), nil
}

func (res *SClusterResourceBase) SetCluster(userCred mcclient.TokenCredential, cls *SCluster) {
	res.ClusterId = cls.GetId()
	res.SyncCloudProjectId(userCred, cls.GetOwnerId())
}

func (res *SClusterResourceBase) GetClusterClient() (*client.ClusterManager, error) {
	cls, err := res.GetCluster()
	if err != nil {
		return nil, err
	}
	return client.GetManagerByCluster(cls)
}

func GetClusterModelObjects(man IClusterModelManager, cluster *SCluster) ([]IClusterModel, error) {
	q := man.Query().Equals("cluster_id", cluster.GetId())
	rows, err := q.Rows()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	ret := make([]IClusterModel, 0)
	defer rows.Close()
	for rows.Next() {
		m, err := db.NewModelObject(man)
		if err != nil {
			return nil, errors.Wrapf(err, "NewModelObject of %s", man.Keyword())
		}
		if err := q.Row2Struct(rows, m); err != nil {
			return nil, errors.Wrapf(err, "Row2Struct of %s", man.Keyword())
		}
		ret = append(ret, m.(IClusterModel))
	}
	return ret, nil
}

func SyncClusterResources(
	ctx context.Context,
	man IClusterModelManager,
	userCred mcclient.TokenCredential,
	cluster *SCluster) compare.SyncResult {

	localObjs := make([]db.IModel, 0)
	remoteObjs := make([]interface{}, 0)
	syncResult := compare.SyncResult{}

	clsCli, err := client.GetManagerByCluster(cluster)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "Get cluster %s client", cluster.GetName()))
		return syncResult
	}

	listObjs, err := man.ListRemoteObjects(clsCli)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}
	dbObjs, err := GetClusterModelObjects(man, cluster)
	if err != nil {
		syncResult.Error(errors.Wrapf(err, "get %s db objects", man.Keyword()))
		return syncResult
	}

	for i := range dbObjs {
		dbObj := dbObjs[i]
		if taskman.TaskManager.IsInTask(dbObj) {
			log.Warningf("cluster %s resource %s object %s is in task, exit this sync task", dbObj.GetClusterId(), dbObj.Keyword(), dbObj.GetName())
			syncResult.Error(fmt.Errorf("object %s is in task", dbObjs[i].GetName()))
			return syncResult
		}
	}

	removed := make([]IClusterModel, 0)
	commondb := make([]IClusterModel, 0)
	commonext := make([]interface{}, 0)
	added := make([]interface{}, 0)

	getGlobalIdF := func(obj interface{}) string {
		return man.GetRemoteObjectGlobalId(cluster, obj)
	}

	if err := CompareRemoteObjectSets(
		dbObjs, listObjs,
		getGlobalIdF,
		&removed, &commondb, &commonext, &added); err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i += 1 {
		if err := SyncRemovedClusterResource(ctx, userCred, removed[i]); err != nil {
			syncResult.DeleteError(err)
		} else {
			syncResult.Delete()
		}
	}

	for i := 0; i < len(commondb); i += 1 {
		if err := SyncUpdatedClusterResource(ctx, userCred, man, commondb[i], commonext[i]); err != nil {
			syncResult.UpdateError(err)
		} else {
			localObjs = append(localObjs, commondb[i])
			remoteObjs = append(remoteObjs, commonext[i])
			syncResult.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		newObj, err := NewFromRemoteObject(ctx, userCred, man, cluster, added[i])
		if err != nil {
			syncResult.AddError(errors.Wrapf(err, "add object"))
		} else {
			localObjs = append(localObjs, newObj)
			remoteObjs = append(remoteObjs, added[i])
			syncResult.Add()
		}
	}
	return syncResult
}

func NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IClusterModelManager,
	cluster *SCluster,
	obj interface{}) (db.IModel, error) {
	lockman.LockClass(ctx, man, db.GetLockClassKey(man, userCred))
	defer lockman.ReleaseClass(ctx, man, db.GetLockClassKey(man, userCred))

	localObj, exist, err := man.IsRemoteObjectLocalExist(userCred, cluster, obj)
	if err != nil {
		return nil, errors.Wrap(err, "check IsRemoteObjectLocalExist")
	}
	if exist {
		return nil, httperrors.NewDuplicateResourceError("%s %v already exists", man.Keyword(), localObj.GetName())
	}
	dbObj, err := man.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, errors.Wrapf(err, "NewFromRemoteObject %s", man.Keyword())
	}
	if err := man.TableSpec().Insert(ctx, dbObj); err != nil {
		return nil, errors.Wrapf(err, "Insert %#v to database", dbObj)
	}
	if err := dbObj.UpdateFromRemoteObject(ctx, userCred, obj); err != nil {
		return nil, errors.Wrap(err, "UpdateFromRemoteObject")
	}
	return dbObj, nil
}

func SyncRemovedClusterResource(ctx context.Context, userCred mcclient.TokenCredential, dbObj IClusterModel) error {
	lockman.LockObject(ctx, dbObj)
	defer lockman.ReleaseObject(ctx, dbObj)

	if err := dbObj.ValidateDeleteCondition(ctx); err != nil {
		err := errors.Wrapf(err, "ValidateDeleteCondition")
		dbObj.SetStatus(userCred, api.ClusterResourceStatusDeleteFail, err.Error())
		return err
	}

	if err := db.CustomizeDelete(dbObj, ctx, userCred, nil, nil); err != nil {
		err := errors.Wrap(err, "CustomizeDelete")
		dbObj.SetStatus(userCred, api.ClusterStatusDeleteFail, err.Error())
		return err
	}

	if err := dbObj.Delete(ctx, userCred); err != nil {
		err := errors.Wrapf(err, "Delete")
		dbObj.SetStatus(userCred, api.ClusterResourceStatusDeleteFail, err.Error())
		return err
	}
	dbObj.PostDelete(ctx, userCred)
	return nil
}

func SyncUpdatedClusterResource(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	man IClusterModelManager,
	dbObj IClusterModel, extObj interface{}) error {
	diff, err := db.UpdateWithLock(ctx, dbObj, func() error {
		if err := dbObj.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
			return err
		}
		cls, err := dbObj.GetCluster()
		if err != nil {
			return err
		}
		dbObj.SetExternalId(man.GetRemoteObjectGlobalId(cls, extObj))
		return nil
	})
	if err != nil {
		log.Errorf("Update from remote object error: %v", err)
		return err
	}
	db.OpsLog.LogSyncUpdate(dbObj, diff, userCred)
	return nil
}

func (m *SClusterResourceBaseManager) ListRemoteObjects(clsCli *client.ClusterManager) ([]interface{}, error) {
	resInfo := m.GetK8SResourceInfo()
	k8sCli := clsCli.GetHandler()
	k8sObjs, err := k8sCli.List(resInfo.ResourceName, "", labels.Everything().String())
	if err != nil {
		return nil, errors.Wrapf(err, "list k8s %s remote objects", resInfo.KindName)
	}
	ret := make([]interface{}, len(k8sObjs))
	for i := range k8sObjs {
		ret[i] = k8sObjs[i]
	}
	return ret, nil
}

func (m *SClusterResourceBaseManager) GetRemoteObjectGlobalId(cluster *SCluster, obj interface{}) string {
	return string(obj.(metav1.Object).GetUID())
}

func (m *SClusterResourceBaseManager) IsRemoteObjectLocalExist(userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, bool, error) {
	metaObj := obj.(metav1.Object)
	if localObj, _ := m.GetByName(userCred, cluster.GetId(), metaObj.GetName()); localObj != nil {
		return localObj, true, nil
	}
	return nil, false, nil
}

func (m *SClusterResourceBaseManager) NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	obj interface{},
) (IClusterModel, error) {
	dbObj, err := db.NewModelObject(m)
	if err != nil {
		return nil, errors.Wrap(err, "NewModelObject")
	}
	metaObj := obj.(metav1.Object)
	dbObj.(db.IExternalizedModel).SetExternalId(m.GetRemoteObjectGlobalId(cluster, obj))
	dbObj.(IClusterModel).SetName(metaObj.GetName())
	dbObj.(IClusterModel).SetCluster(userCred, cluster)
	return dbObj.(IClusterModel), nil
}

func (obj *SClusterResourceBase) GetClusterId() string {
	return obj.ClusterId
}

func (obj *SClusterResourceBase) SetName(name string) {
	obj.Name = name
}

func (obj *SClusterResourceBase) SetIsSystem(isSystem bool) {
	obj.IsSystem = isSystem
}

func (obj *SClusterResourceBase) UpdateFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	extObj interface{}) error {
	metaObj := extObj.(metav1.Object)
	resVersion := metaObj.GetResourceVersion()
	ver, _ := strconv.ParseInt(resVersion, 10, 32)
	var curResVer int64
	if obj.ResourceVersion != "" {
		curResVer, _ = strconv.ParseInt(obj.ResourceVersion, 10, 32)
	}
	if ver < curResVer {
		return errors.Errorf("remote object resourceVersion less than local: %d < %d", ver, curResVer)
	}
	if obj.GetName() != metaObj.GetName() {
		obj.SetName(metaObj.GetName())
	}
	if obj.ResourceVersion != resVersion {
		obj.ResourceVersion = resVersion
	}
	return nil
}

func (m *SClusterResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ClusterResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Cluster != "" {
		cls, err := ClusterManager.FetchClusterByIdOrName(userCred, input.Cluster)
		if err != nil {
			return nil, err
		}
		input.Cluster = cls.GetId()
		q = q.Equals("cluster_id", cls.GetId())
	}
	return q, nil
}

func (m *SClusterResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ClusterResourceDetail {
	rows := make([]api.ClusterResourceDetail, len(objs))
	vRows := m.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	k8sResInfo := m.GetK8SResourceInfo()
	for i := range vRows {
		detail := api.ClusterResourceDetail{
			VirtualResourceDetails: vRows[i],
		}
		resObj := objs[i].(IClusterModel)
		cls, err := resObj.GetCluster()
		if err != nil {
			log.Errorf("Get resource %s cluster error: %v", resObj.GetId(), err)
		} else {
			detail.Cluster = cls.GetName()
			detail.ClusterId = cls.GetId()
			detail.ClusterID = cls.GetId()
		}
		if k8sResInfo.Object != nil {
			detail, err = m.GetK8SResourceMetaDetail(resObj, detail)
			if err != nil {
				log.Errorf("Get resource %s k8s meta error: %v", resObj.GetId(), err)
			}
		}
		rows[i] = detail
	}
	return rows
}

func GetK8SObject(res IClusterModel) (runtime.Object, error) {
	cli, err := res.GetClusterClient()
	man := res.GetModelManager().(IClusterModelManager)
	info := man.GetK8SResourceInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "get object %s/%s kubernetes client", info.ResourceName, res.GetName())
	}
	namespaceName := ""
	if nsResObj, ok := res.(INamespaceModel); ok {
		nsObj, err := nsResObj.GetNamespace()
		if err != nil {
			return nil, errors.Wrapf(err, "get object %s/%s local db namespace", info.ResourceName, res.GetName())
		}
		namespaceName = nsObj.GetName()
	}
	k8sObj, err := cli.GetHandler().Get(info.ResourceName, namespaceName, res.GetName())
	if err != nil {
		return nil, errors.Wrapf(err, "get object from k8s %s/%s", info.ResourceName, res.GetName())
	}
	return k8sObj, nil
}

func (m *SClusterResourceBaseManager) GetK8SResourceMetaDetail(obj IClusterModel, detail api.ClusterResourceDetail) (api.ClusterResourceDetail, error) {
	k8sObj, err := GetK8SObject(obj)
	if err != nil {
		return detail, errors.Wrap(err, "get object from k8s")
	}
	metaObj := k8sObj.(metav1.Object)
	detail.ClusterK8SResourceMetaDetail = &api.ClusterK8SResourceMetaDetail{
		TypeMeta:                   GetK8SObjectTypeMeta(k8sObj),
		ResourceVersion:            metaObj.GetResourceVersion(),
		Generation:                 metaObj.GetGeneration(),
		CreationTimestamp:          metaObj.GetCreationTimestamp().Time,
		DeletionGracePeriodSeconds: metaObj.GetDeletionGracePeriodSeconds(),
		Labels:                     metaObj.GetLabels(),
		Annotations:                metaObj.GetAnnotations(),
		OwnerReferences:            metaObj.GetOwnerReferences(),
		Finalizers:                 metaObj.GetFinalizers(),
	}
	if deletionTimestamp := metaObj.GetDeletionTimestamp(); deletionTimestamp != nil {
		detail.ClusterK8SResourceMetaDetail.DeletionTimestamp = &deletionTimestamp.Time
	}
	return detail, nil
}

// GetExtraDetails is deprecated
func (res *SClusterResourceBase) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.ClusterResourceDetail, error) {
	return api.ClusterResourceDetail{}, nil
}

func (obj *SClusterResourceBase) StartCreateTask(resObj IClusterModel, ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query, data jsonutils.JSONObject) {
	obj.SVirtualResourceBase.PostCreate(ctx, userCred, ownerCred, query, data)
	if err := StartClusterResourceCreateTask(ctx, userCred, resObj, data.(*jsonutils.JSONDict), ""); err != nil {
		log.Errorf("Create %s resource task error: %v", obj.Keyword(), err)
	}
}

func StartClusterResourceCreateTask(ctx context.Context, userCred mcclient.TokenCredential, res IClusterModel, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceCreateTask", res, userCred, data, parentId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func CreateRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, man IClusterModelManager, model IClusterModel, data jsonutils.JSONObject) (interface{}, error) {
	cli, err := model.GetClusterClient()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster client")
	}
	obj, err := man.NewRemoteObjectForCreate(model, cli, data)
	if err != nil {
		return nil, errors.Wrap(err, "NewRemoteObjectForCreate")
	}
	obj, err = man.CreateRemoteObject(model, cli, obj)
	if err != nil {
		return nil, errors.Wrap(err, "CreateRemoteObject")
	}
	if err := model.UpdateFromRemoteObject(ctx, userCred, obj); err != nil {
		return nil, errors.Wrap(err, "UpdateFromRemoteObject after CreateRemoteObject")
	}
	cls, err := model.GetCluster()
	if err != nil {
		return nil, errors.Wrap(err, "get cluster")
	}
	if _, err := db.Update(model, func() error {
		model.SetExternalId(man.GetRemoteObjectGlobalId(cls, obj))
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "set external id")
	}
	return obj, nil
}

func (m *SClusterResourceBaseManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	return nil, fmt.Errorf("NewRemoteObjectForCreate of %s not override", m.kindName)
}

func (m *SClusterResourceBaseManager) CreateRemoteObject(_ IClusterModel, cli *client.ClusterManager, obj interface{}) (interface{}, error) {
	metaObj := obj.(metav1.Object)
	return cli.GetHandler().CreateV2(m.resourceName, metaObj.GetNamespace(), obj.(runtime.Object))
}

func DeleteRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, man IClusterModelManager, model IClusterModel, data jsonutils.JSONObject) error {
	cli, err := model.GetClusterClient()
	if err != nil {
		return errors.Wrap(err, "get cluster client")
	}
	if err := model.DeleteRemoteObject(cli); err != nil {
		return errors.Wrap(err, "DeleteRemoteObject")
	}
	return nil
}

func (m *SClusterResourceBaseManager) GetClusterModelManager() IClusterModelManager {
	return m.GetIModelManager().(IClusterModelManager)
}

func (obj *SClusterResourceBase) GetClusterModelManager() IClusterModelManager {
	return obj.GetModelManager().(IClusterModelManager)
}

func (obj *SClusterResourceBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Resource %s delete do nothing", obj.Keyword())
	return nil
}

func (obj *SClusterResourceBase) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SVirtualResourceBase.Delete(ctx, userCred)
}

func (obj *SClusterResourceBase) DeleteRemoteObject(cli *client.ClusterManager) error {
	resInfo := obj.GetClusterModelManager().GetK8SResourceInfo()
	if err := cli.GetHandler().Delete(resInfo.ResourceName, "", obj.GetName(), &metav1.DeleteOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func StartClusterResourceDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, res IClusterModel, data *jsonutils.JSONDict, parentId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterResourceDeleteTask", res, userCred, data, parentId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (obj *SClusterResourceBase) StartDeleteTask(res IClusterModel, ctx context.Context, userCred mcclient.TokenCredential) {
	if err := StartClusterResourceDeleteTask(ctx, userCred, res, jsonutils.NewDict(), ""); err != nil {
		log.Errorf("StartClusterResourceDeleteTask error: %v", err)
	}
}

func (m *SClusterResourceBaseManager) OnRemoteObjectCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, obj runtime.Object) {
	resMan := m.GetClusterModelManager()
	OnRemoteObjectCreate(resMan, ctx, userCred, cluster, obj)
}

func (m *SClusterResourceBaseManager) OnRemoteObjectUpdate(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, oldObj, newObj runtime.Object) {
	resMan := m.GetClusterModelManager()
	metaObj := newObj.(metav1.Object)
	objName := metaObj.GetName()
	dbObj, err := m.GetByName(userCred, cluster.GetId(), objName)
	if err != nil {
		log.Errorf("OnRemoteObjectUpdate get %s local object %s error: %v", resMan.Keyword(), objName, err)
		return
	}
	OnRemoteObjectUpdate(resMan, ctx, userCred, dbObj, newObj)
}

func (m *SClusterResourceBaseManager) OnRemoteObjectDelete(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, obj runtime.Object) {
	resMan := m.GetClusterModelManager()
	metaObj := obj.(metav1.Object)
	objName := metaObj.GetName()
	dbObj, err := m.GetByName(userCred, cluster.GetId(), objName)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// local object already deleted
			return
		}
		log.Errorf("OnRemoteObjectDelete get %s local object %s error: %v", resMan.Keyword(), objName, err)
		return
	}
	OnRemoteObjectDelete(resMan, ctx, userCred, dbObj)
}

func OnRemoteObjectCreate(resMan IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, obj runtime.Object) {
	objName := obj.(metav1.Object).GetName()
	localObj, exist, err := resMan.IsRemoteObjectLocalExist(userCred, cluster.(*SCluster), obj)
	if err != nil {
		log.Errorf("check IsRemoteObjectLocalExist error: %v", err)
		return
	}
	if exist {
		// update localObj is already exists
		OnRemoteObjectUpdate(resMan, ctx, userCred, localObj, obj)
	} else {
		// create localObj
		log.Debugf("cluster %s remote object %s/%s created, sync to local", cluster.GetName(), resMan.Keyword(), objName)
		if _, err := NewFromRemoteObject(ctx, userCred, resMan, cluster.(*SCluster), obj); err != nil {
			log.Errorf("NewFromRemoteObject for %s error: %v", resMan.Keyword(), err)
			return
		}
	}
}

func OnRemoteObjectUpdate(resMan IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential, dbObj IClusterModel, newObj runtime.Object) {
	log.Debugf("remote object %s/%s update, sync to local", resMan.Keyword(), dbObj.GetName())
	if err := SyncUpdatedClusterResource(ctx, userCred, resMan, dbObj, newObj); err != nil {
		log.Errorf("OnRemoteObjectUpdate SyncUpdatedClusterResource %s %s error: %v", resMan.Keyword(), dbObj.GetName(), err)
		return
	}
}

func OnRemoteObjectDelete(resMan IClusterModelManager, ctx context.Context, userCred mcclient.TokenCredential, dbObj IClusterModel) {
	log.Infof("remote object %s/%s deleted, delete local", resMan.Keyword(), dbObj.GetName())
	if err := SyncRemovedClusterResource(ctx, userCred, dbObj); err != nil {
		log.Errorf("OnRemoteObjectDelete %s %s SyncRemovedClusterResource error: %v", resMan.Keyword(), dbObj.GetName(), err)
		return
	}
}

func GetResourcesByClusters(man IClusterModelManager, clusterIds []string, ret interface{}) error {
	q := man.Query().In("cluster_id", clusterIds)
	if err := q.All(ret); err != nil {
		return errors.Wrapf(err, "fetch %s resources by clusters")
	}
	return nil
}
