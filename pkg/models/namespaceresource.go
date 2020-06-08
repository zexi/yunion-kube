package models

import (
	"context"
	"database/sql"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models/manager"
)

type SNamespaceResourceBaseManager struct {
	SClusterResourceBaseManager
}

type SNamespaceResourceBase struct {
	SClusterResourceBase

	NamespaceId string `width:"36" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

func NewNamespaceResourceBaseManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
	resName string,
	kind string,
	object runtime.Object) SNamespaceResourceBaseManager {
	return SNamespaceResourceBaseManager{
		SClusterResourceBaseManager: NewClusterResourceBaseManager(dt, tableName, keyword, keywordPlural, resName, kind, object),
	}
}

func (r *SNamespaceResourceBase) GetParentId() string {
	return r.NamespaceId
}

func (m *SNamespaceResourceBaseManager) GetByIdOrName(userCred mcclient.IIdentityProvider, clusterId, namespaceId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByIdOrName(m, userCred, clusterId, namespaceId, resId)
}

func (m *SNamespaceResourceBaseManager) GetByName(userCred mcclient.IIdentityProvider, clusterId, namespaceId string, resId string) (IClusterModel, error) {
	return FetchClusterResourceByName(m, userCred, clusterId, namespaceId, resId)
}

func (m SNamespaceResourceBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.NamespaceResourceCreateInput) (*api.NamespaceResourceCreateInput, error) {
	cData, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &data.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.ClusterResourceCreateInput = *cData

	if data.Namespace == "" {
		return nil, httperrors.NewNotEmptyError("namespace is empty")
	}
	nsObj, err := NamespaceManager.GetByIdOrName(userCred, data.Cluster, data.Namespace)
	if err != nil {
		return nil, NewCheckIdOrNameError("namespace", data.Namespace, err)
	}
	data.Namespace = nsObj.GetId()

	return data, nil
}

func (res *SNamespaceResourceBase) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := res.SClusterResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	input := new(api.NamespaceResourceCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "namespace resource unmarshal data")
	}
	res.NamespaceId = input.Namespace
	return nil
}

func (res *SNamespaceResourceBase) GetNamespace() (*SNamespace, error) {
	obj, err := NamespaceManager.FetchById(res.NamespaceId)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch namespace %s", res.NamespaceId)
	}
	return obj.(*SNamespace), nil
}

type INamespaceModel interface {
	IClusterModel

	GetNamespace() (*SNamespace, error)
	SetNamespace(userCred mcclient.TokenCredential, ns *SNamespace)
}

func (m *SNamespaceResourceBaseManager) IsRemoteObjectLocalExist(userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, bool, error) {
	metaObj := obj.(metav1.Object)
	objName := metaObj.GetName()
	objNs := metaObj.GetNamespace()
	localNs, err := NamespaceManager.GetByName(userCred, cluster.GetId(), objNs)
	if err != nil {
		return nil, false, errors.Wrapf(err, "get namespace %s", objNs)
	}
	if localObj, _ := m.GetByName(userCred, cluster.GetId(), localNs.GetId(), objName); localObj != nil {
		return localObj, true, nil
	}
	return nil, false, nil
}

func (res *SNamespaceResourceBaseManager) NewFromRemoteObject(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster *SCluster,
	remoteObj interface{}) (IClusterModel, error) {
	clsObj, err := res.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, remoteObj)
	if err != nil {
		return nil, errors.Wrap(err, "call cluster resource base NewFromRemoteObject")
	}
	localObj := clsObj.(INamespaceModel)
	ns := remoteObj.(metav1.Object).GetNamespace()
	localNs, err := NamespaceManager.GetByName(userCred, cluster.GetId(), ns)
	if err != nil {
		return nil, errors.Wrapf(err, "get local namespace by name %s", ns)
	}
	localObj.SetNamespace(userCred, localNs.(*SNamespace))
	return localObj, nil
}

func (m *SNamespaceResourceBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.ClusterResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Namespace != "" {
		ns, err := NamespaceManager.GetByIdOrName(userCred, input.Cluster, input.Namespace)
		if err != nil {
			return nil, err
		}
		q = q.Equals("namespace_id", ns.GetId())
	}
	return q, nil
}

func (res *SNamespaceResourceBase) SetNamespace(userCred mcclient.TokenCredential, ns *SNamespace) {
	res.NamespaceId = ns.GetId()
}

func (m *SNamespaceResourceBaseManager) FilterByHiddenSystemAttributes(q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	q = m.SStatusStandaloneResourceBaseManager.FilterBySystemAttributes(q, userCred, query, scope)
	input := new(api.NamespaceResourceListInput)
	if err := query.Unmarshal(input); err != nil {
		panic(fmt.Sprintf("unmarshal namespace resource list input error: %v", err))
	}
	isSystem := false
	if input.System != nil {
		var isAllow bool
		if consts.IsRbacEnabled() {
			allowScope := policy.PolicyManager.AllowScope(userCred, consts.GetServiceType(), m.KeywordPlural(), policy.PolicyActionList, "system")
			if !scope.HigherThan(allowScope) {
				isAllow = true
			}
		} else {
			if userCred.HasSystemAdminPrivilege() {
				isAllow = true
			}
		}
		if !isAllow {
			isSystem = false
		}
	}
	nsQ := NamespaceManager.Query("id")
	nsSq := nsQ.Equals("name", userCred.GetProjectId()).IsTrue("is_system")
	if !isSystem {
		q = q.Filter(
			sqlchemy.OR(
				sqlchemy.In(q.Field("namespace_id"), nsSq.SubQuery()),
				sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))),
			),
		)
	}
	return q
}

func (m *SNamespaceResourceBaseManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NamespaceResourceDetail {
	rows := make([]api.NamespaceResourceDetail, len(objs))
	cRows := m.SClusterResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range cRows {
		detail := api.NamespaceResourceDetail{
			ClusterResourceDetail: cRows[i],
		}
		resObj := objs[i].(INamespaceModel)
		ns, err := resObj.GetNamespace()
		if err != nil {
			log.Errorf("Get resource %s namespace error: %v", resObj.GetId(), err)
		} else {
			detail.Namespace = ns.GetName()
			detail.NamespaceId = ns.GetId()
		}
		rows[i] = detail
	}
	return rows
}

func (res *SNamespaceResourceBase) PostDelete(obj INamespaceModel, ctx context.Context, userCred mcclient.TokenCredential) {
	res.SClusterResourceBase.PostDelete(ctx, userCred)
	res.StartDeleteTask(obj, ctx, userCred)
}

func (res *SNamespaceResourceBase) DeleteRemoteObject(cli *client.ClusterManager) error {
	resInfo := res.GetClusterModelManager().GetK8SResourceInfo()
	ns, err := res.GetNamespace()
	if err != nil {
		return errors.Wrap(err, "get namespace")
	}
	if err := cli.GetHandler().Delete(resInfo.ResourceName, ns.GetName(), res.GetName(), &metav1.DeleteOptions{}); err != nil {
		if kerrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// OnRemoteObjectCreate invoked when remote object created
func (m *SNamespaceResourceBaseManager) OnRemoteObjectCreate(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, obj runtime.Object) {
	m.SClusterResourceBaseManager.OnRemoteObjectCreate(ctx, userCred, cluster, obj)
}

// OnRemoteObjectUpdate invoked when remote resource updated
func (m *SNamespaceResourceBaseManager) OnRemoteObjectUpdate(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, oldObj, newObj runtime.Object) {
	resMan := m.GetClusterModelManager()
	metaObj := newObj.(metav1.Object)
	objName := metaObj.GetName()
	objNamespace := metaObj.GetNamespace()
	dbNs, err := NamespaceManager.GetByName(userCred, cluster.GetId(), objNamespace)
	if err != nil {
		log.Errorf("OnRemoteObjectUpdate for %s get namespace error: %v", resMan.Keyword(), err)
		return
	}
	dbObj, err := m.GetByName(userCred, cluster.GetId(), dbNs.GetId(), objName)
	if err != nil {
		log.Errorf("OnRemoteObjectUpdate get %s local object %s error: %v", resMan.Keyword(), objName, err)
		return
	}
	OnRemoteObjectUpdate(resMan, ctx, userCred, dbObj, newObj)
}

// OnRemoteObjectDelete invoked when remote resource deleted
func (m *SNamespaceResourceBaseManager) OnRemoteObjectDelete(ctx context.Context, userCred mcclient.TokenCredential, cluster manager.ICluster, obj runtime.Object) {
	resMan := m.GetClusterModelManager()
	metaObj := obj.(metav1.Object)
	objName := metaObj.GetName()
	objNamespace := metaObj.GetNamespace()
	dbNs, err := NamespaceManager.GetByName(userCred, cluster.GetId(), objNamespace)
	if err != nil {
		log.Errorf("OnRemoteObjectDelete for %s get namespace error: %v", resMan.Keyword(), err)
		return
	}
	dbObj, err := m.GetByName(userCred, cluster.GetId(), dbNs.GetId(), objName)
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
