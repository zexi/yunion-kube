package models

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"k8s.io/kubernetes/pkg/apis/rbac/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	RoleBindingManager *SRoleBindingManager
	_                  db.IModel = new(SRoleBinding)
)

func init() {
	RoleBindingManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SRoleBindingManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SRoleBinding),
				"rolebindings_tbl",
				"rbacrolebinding",
				"rbacrolebindings",
				api.ResourceNameRoleBinding,
				api.KindNameRoleBinding,
				new(rbac.RoleBinding),
			),
		}
	}).(*SRoleBindingManager)
}

type SRoleBindingManager struct {
	SNamespaceResourceBaseManager
	SRoleRefResourceBaseManager
}

type SRoleBinding struct {
	SNamespaceResourceBase
	SRoleRefResourceBase
}

func (m *SRoleBindingManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
	if err != nil {
		return nil, err
	}
	return q, nil
}

func (m *SRoleBindingManager) ValidateRoleBinding(rb *rbacv1.RoleBinding) error {
	return ValidateK8sObject(rb, new(rbac.RoleBinding), func(out interface{}) field.ErrorList {
		return validation.ValidateRoleBinding(out.(*rbac.RoleBinding))
	})
}

func (m *SRoleBindingManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.RoleBindingCreateInput) (*api.RoleBindingCreateInput, error) {
	nInput, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.NamespaceResourceCreateInput = *nInput

	var roleMan IRoleBaseManager
	if input.RoleRef.Kind == api.KindNameRole {
		roleMan = GetRoleManager()
	} else if input.RoleRef.Kind == api.KindNameClusterRole {
		roleMan = GetClusterRoleManager()
	} else {
		return nil, httperrors.NewNotAcceptableError("not support role ref kind %s", input.RoleRef.Kind)
	}
	if err := m.SRoleRefResourceBaseManager.ValidateRoleRef(roleMan, userCred, input.RoleRef); err != nil {
		return nil, err
	}

	rb := input.ToRoleBinding()
	if err := m.ValidateRoleBinding(rb); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SRoleBindingManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func (rb *SRoleBinding) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := rb.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	return nil
}
