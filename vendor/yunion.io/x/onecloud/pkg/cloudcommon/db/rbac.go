package db

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

func isListRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	return isListRbacAllowedInternal(manager, manager.KeywordPlural(), userCred, isAdminMode)
}

func isListRbacAllowedInternal(manager IModelManager, resource string, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	var requireAdmin bool
	var ownerId string
	if userCred != nil {
		ownerId = manager.GetOwnerId(userCred)
	}
	if len(ownerId) > 0 {
		if isAdminMode {
			requireAdmin = true
		} else {
			requireAdmin = false
		}
	} else {
		requireAdmin = true
	}
	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		resource, policy.PolicyActionList)

	switch {
	case result == rbacutils.GuestAllow:
		return true
	case result == rbacutils.UserAllow && userCred != nil && userCred.IsValid():
		return true
	case result == rbacutils.OwnerAllow && !requireAdmin:
		return true
	}

	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		resource, policy.PolicyActionList)

	return result == rbacutils.AdminAllow
}

func isJointListRbacAllowed(manager IJointModelManager, userCred mcclient.TokenCredential, isAdminMode bool) bool {
	return isListRbacAllowedInternal(manager.GetMasterManager(), manager.KeywordPlural(), userCred, isAdminMode)
}

func isClassActionRbacAllowed(manager IModelManager, userCred mcclient.TokenCredential, ownerProjId string, action string, extra ...string) bool {
	var requireAdmin bool
	var ownerId string
	if userCred != nil {
		ownerId = manager.GetOwnerId(userCred)
	}
	if len(ownerId) > 0 {
		if ownerProjId == ownerId {
			requireAdmin = false
		} else {
			requireAdmin = true
		}
	} else {
		requireAdmin = true
	}

	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	switch {
	case result == rbacutils.GuestAllow:
		return true
	case result == rbacutils.UserAllow && userCred != nil && userCred.IsValid():
		return true
	case result == rbacutils.OwnerAllow && !requireAdmin:
		return true
	}

	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	return result == rbacutils.AdminAllow
}

func isObjectRbacAllowed(manager IModelManager, model IModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	var requireAdmin bool
	var isOwner bool

	var ownerId string

	if userCred != nil {
		ownerId = model.GetModelManager().GetOwnerId(userCred)
	}

	if len(ownerId) > 0 {
		objOwnerId := model.GetOwnerProjectId()
		if ownerId == objOwnerId || model.IsSharable() {
			isOwner = true
			requireAdmin = false
		} else {
			isOwner = false
			requireAdmin = true
		}
	} else {
		requireAdmin = true
	}

	result := policy.PolicyManager.Allow(false, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	switch {
	case result == rbacutils.GuestAllow:
		return true
	case result == rbacutils.UserAllow && userCred != nil && userCred.IsValid():
		return true
	case result == rbacutils.OwnerAllow && isOwner && !requireAdmin:
		return true
	}

	result = policy.PolicyManager.Allow(true, userCred, consts.GetServiceType(),
		manager.KeywordPlural(), action, extra...)
	return result == rbacutils.AdminAllow
}

func isJointObjectRbacAllowed(manager IJointModelManager, item IJointModel, userCred mcclient.TokenCredential, action string, extra ...string) bool {
	return isObjectRbacAllowed(manager, item.Master(), userCred, action, extra...)
}

func IsAdminAllowList(userCred mcclient.TokenCredential, manager IModelManager) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionList)
}

func IsAdminAllowCreate(userCred mcclient.TokenCredential, manager IModelManager) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionCreate)
}

func IsAdminAllowClassPerform(userCred mcclient.TokenCredential, manager IModelManager, action string) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), manager.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowGet(userCred mcclient.TokenCredential, obj IModel) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet)
}

func IsAdminAllowGetSpec(userCred mcclient.TokenCredential, obj IModel, spec string) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionGet, spec)
}

func IsAdminAllowPerform(userCred mcclient.TokenCredential, obj IModel, action string) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionPerform, action)
}

func IsAdminAllowUpdate(userCred mcclient.TokenCredential, obj IModel) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionUpdate)
}

func IsAdminAllowDelete(userCred mcclient.TokenCredential, obj IModel) bool {
	return userCred.IsAdminAllow(consts.GetServiceType(), obj.KeywordPlural(), policy.PolicyActionDelete)
}
