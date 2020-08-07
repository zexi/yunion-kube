package models

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

type SRoleRefResourceBaseManager struct{}

type SRoleRefResourceBase struct {
	RefKind string `width:"36" charset:"ascii" nullable:"false" index:"true" list:"user"`
	RefId   string `width:"36" charset:"ascii" nullable:"false" index:"true" list:"user"`
}

type IRoleBaseManager interface {
	db.IModelManager
	GetRoleKind() string
}

func (m *SRoleRefResourceBaseManager) ValidateRoleRef(roleObjManager IRoleBaseManager, userCred mcclient.TokenCredential, ref *api.RoleRef) error {
	if ref == nil {
		return httperrors.NewNotEmptyError("role_ref must provided")
	}
	kind := roleObjManager.GetRoleKind()
	if ref.Kind != kind {
		return httperrors.NewNotAcceptableError("role reference kind must %s, input %s", kind, ref.Kind)
	}
	refObj, err := roleObjManager.FetchByIdOrName(userCred, ref.Id)
	if err != nil {
		return err
	}
	ref.Id = refObj.GetId()
	ref.Name = refObj.GetName()
	return nil
}
