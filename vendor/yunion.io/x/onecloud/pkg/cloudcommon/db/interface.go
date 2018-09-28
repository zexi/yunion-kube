package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

type IModelManager interface {
	lockman.ILockedClass

	GetContextManager() []IModelManager

	// Table() *sqlchemy.STable
	TableSpec() *sqlchemy.STableSpec

	// Keyword() string
	KeywordPlural() string
	Alias() string
	AliasPlural() string

	ValidateName(name string) error

	// list hooks
	AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	ValidateListConditions(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error)
	ExtraSearchConditions(ctx context.Context, q *sqlchemy.SQuery, like string) []sqlchemy.ICondition

	// fetch hook
	Query(val ...string) *sqlchemy.SQuery

	FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery
	FilterByNotId(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery
	FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery
	FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery

	GetOwnerId(userCred mcclient.TokenCredential) string

	// RawFetchById(idStr string) (IModel, error)
	FetchById(idStr string) (IModel, error)
	FetchByName(ownerProjId string, idStr string) (IModel, error)
	FetchByIdOrName(ownerProjId string, idStr string) (IModel, error)

	// create hooks
	AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	OnCreateComplete(ctx context.Context, items []IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)

	// allow perform action
	AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	InitializeData() error
}

type IModel interface {
	lockman.ILockedObject

	GetName() string

	GetModelManager() IModelManager
	SetModelManager(IModelManager)

	GetShortDesc() *jsonutils.JSONDict

	// list hooks
	GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict

	// get hooks
	AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool
	GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict

	// create hooks
	CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) error
	PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject)

	// allow perform action
	AllowPerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	PerformAction(ctx context.Context, userCred mcclient.TokenCredential, action string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	// update hooks
	ValidateUpdateCondition(ctx context.Context) error

	AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool
	ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error)
	PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)
	PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)

	// delete hooks
	AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool
	ValidateDeleteCondition(ctx context.Context) error
	CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error
	PreDelete(ctx context.Context, userCred mcclient.TokenCredential)
	MarkDelete() error
	Delete(ctx context.Context, userCred mcclient.TokenCredential) error
	PostDelete(ctx context.Context, userCred mcclient.TokenCredential)

	GetOwnerProjectId() string
}

type IResourceModelManager interface {
	IModelManager
}

type IResourceModel interface {
	IModel
}

type IJointModelManager interface {
	IResourceModelManager

	GetMasterManager() IStandaloneModelManager
	GetSlaveManager() IStandaloneModelManager

	FetchByIds(masterId string, slaveId string) (IJointModel, error)

	AllowListDescendent(ctx context.Context, userCred mcclient.TokenCredential, model IStandaloneModel, query jsonutils.JSONObject) bool
	AllowAttach(ctx context.Context, userCred mcclient.TokenCredential, master IStandaloneModel, slave IStandaloneModel) bool
}

type IJointModel interface {
	IResourceModel

	GetJointModelManager() IJointModelManager
	Master() IStandaloneModel
	Slave() IStandaloneModel

	Detach(ctx context.Context, userCred mcclient.TokenCredential) error
	AllowGetJointDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, item IJointModel) bool
	AllowUpdateJointItem(ctx context.Context, userCred mcclient.TokenCredential, item IJointModel) bool
}

type IStandaloneModelManager interface {
	IResourceModelManager
	// GenerateName(ownerProjId string, hint string) string
	// ValidateName(name string) error
	// IsNewNameUnique(name string, projectId string) bool

	FetchByExternalId(idStr string) (IStandaloneModel, error)
}

type IStandaloneModel interface {
	IResourceModel
	// IsAlterNameUnique(name string, projectId string) bool
	// GetExternalId() string
}

type IVirtualModelManager interface {
	IStandaloneModelManager
}

type IVirtualModel interface {
	IStandaloneModel

	IsOwner(userCred mcclient.TokenCredential) bool
	IsAdmin(userCred mcclient.TokenCredential) bool
}

type ISharableVirtualModelManager interface {
	IVirtualModelManager
}

type ISharableVirtualModel interface {
	IVirtualModel
	IsSharable() bool
}

type IAdminSharableVirtualModelManager interface {
	ISharableVirtualModelManager
	GetRecordsSeparator() string
	GetRecordsLimit() int
	ParseInputInfo(data *jsonutils.JSONDict) ([]string, error)
}

type IAdminSharableVirtualModel interface {
	ISharableVirtualModel
	GetInfo() []string
}
