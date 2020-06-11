package models

import (
	"context"
	"database/sql"

	v1 "k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	NamespaceManager *SNamespaceManager
)

func init() {
	NamespaceManager = &SNamespaceManager{
		SClusterResourceBaseManager: NewClusterResourceBaseManager(
			SNamespace{},
			"namespaces_tbl",
			"namespace",
			"namespaces",
			api.ResourceNameNamespace,
			api.KindNameNamespace,
			&v1.Namespace{}),
	}
	NamespaceManager.SetVirtualObject(NamespaceManager)
}

type SNamespaceManager struct {
	SClusterResourceBaseManager
}

type SNamespace struct {
	SClusterResourceBase
}

func (m *SNamespaceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.NamespaceCreateInputV2) (*api.NamespaceCreateInputV2, error) {
	cData, err := m.SClusterResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &data.ClusterResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.ClusterResourceCreateInput = *cData
	return data, nil
}

func (obj *SNamespace) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	if err := obj.SClusterResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data); err != nil {
		return err
	}
	cls, err := obj.GetCluster()
	if err != nil {
		return errors.Wrap(err, "CustomizeCreatData fetch cluster")
	}
	obj.IsSystem = cls.IsSystem
	return nil
}

func (res *SNamespace) SetCluster(userCred mcclient.TokenCredential, cls *SCluster) {
	res.SClusterResourceBase.SetCluster(userCred, cls)
	res.SetIsSystem(cls.IsSystem)
}

func (obj *SNamespaceManager) NewRemoteObjectForCreate(_ IClusterModel, _ *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.NamespaceCreateInputV2)
	data.Unmarshal(input)
	return &v1.Namespace{
		ObjectMeta: input.ToObjectMeta(),
	}, nil
}

func (m *SNamespaceManager) EnsureNamespace(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, cluster *SCluster, data *api.NamespaceCreateInputV2) (*SNamespace, error) {
	data.Cluster = cluster.GetId()
	nsObj, err := m.GetByIdOrName(userCred, cluster.GetId(), data.Name)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			nsModelObj, err := db.DoCreate(m, ctx, userCred, nil, data.JSON(data), ownerCred)
			if err != nil {
				return nil, errors.Wrap(err, "create namespace")
			}
			nsObj = nsModelObj.(IClusterModel)
		} else {
			return nil, errors.Wrapf(err, "get namespace %s", data.Name)
		}
	}
	ns := nsObj.(*SNamespace)
	if err := ns.DoSync(ctx, userCred); err != nil {
		return nil, errors.Wrap(err, "sync to kubernetes cluster")
	}
	return ns, nil
}

func (ns *SNamespace) DoSync(ctx context.Context, userCred mcclient.TokenCredential) error {
	cluster, err := ns.GetCluster()
	if err != nil {
		return errors.Wrapf(err, "get namespace %s cluster", ns.GetName())
	}
	if err := EnsureNamespace(cluster, ns.GetName()); err != nil {
		return errors.Wrapf(err, "ensure namespace %s", ns.GetName())
	}
	rNs, err := GetK8SObject(ns)
	if err != nil {
		return errors.Wrap(err, "get namespace k8s object")
	}
	return SyncUpdatedClusterResource(ctx, userCred, NamespaceManager, ns, rNs)
}

func (m *SNamespaceManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	nsObj, err := m.SClusterResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	return nsObj, nil
}

func (ns *SNamespace) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	ns.SClusterResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	ns.StartCreateTask(ns, ctx, userCred, ownerId, query, data)
}

func (ns *SNamespace) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := ns.SClusterResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	cls, err := ns.GetCluster()
	if err != nil {
		return err
	}
	if ns.IsSystem != cls.IsSystem {
		ns.IsSystem = cls.IsSystem
	}
	k8sNsStatus := string(extObj.(*v1.Namespace).Status.Phase)
	if ns.Status != k8sNsStatus {
		if err := ns.SetStatus(userCred, k8sNsStatus, "update from remote object"); err != nil {
			return err
		}
	}
	return nil
}

func (m *SNamespaceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SClusterResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.ClusterResourceListInput)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SNamespaceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.NamespaceDetailV2 {
	rows := make([]api.NamespaceDetailV2, len(objs))
	cRows := m.SClusterResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range cRows {
		detail := api.NamespaceDetailV2{
			ClusterResourceDetail: cRows[i],
		}
		rows[i] = detail
	}
	return rows
}

func (ns *SNamespace) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if err := ns.SClusterResourceBase.Delete(ctx, userCred); err != nil {
		cluster, err := ns.GetCluster()
		if err != nil {
			return errors.Wrap(err, "get cluster")
		}
		if cluster.IsSystem {
			if !userCred.HasSystemAdminPrivilege() {
				return httperrors.NewForbiddenError("Not system admin")
			}
		}
		return err
	}
	for _, man := range []IClusterModelManager{
		ReleaseManager,
		PodManager,
	} {
		q := man.Query()
		q.Equals("namespace_id", ns.GetId())
		cnt, err := q.CountWithError()
		if err != nil {
			return errors.Wrapf(err, "check %s namespace resource count", man.KeywordPlural())
		}
		if cnt != 0 {
			return httperrors.NewNotAcceptableError("%s has %d resource in namespace %s", man.KeywordPlural(), cnt, ns.GetName())
		}
	}
	return nil
}

func (ns *SNamespace) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	ns.SClusterResourceBase.PostDelete(ctx, userCred)
	ns.StartDeleteTask(ns, ctx, userCred)
}
