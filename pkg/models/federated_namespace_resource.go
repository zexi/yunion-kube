package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

type IFederatedNamespaceModel interface {
	IFederatedModel

	GetFedNamespace() (*SFederatedNamespace, error)
}

// +onecloud:swagger-gen-ignore
type SFederatedNamespaceResourceManager struct {
	SFederatedResourceBaseManager
}

type SFederatedNamespaceResource struct {
	SFederatedResourceBase

	FederatednamespaceId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
}

func NewFedNamespaceResourceManager(
	dt interface{},
	tableName string,
	keyword string,
	keywordPlural string,
) SFederatedNamespaceResourceManager {
	return SFederatedNamespaceResourceManager{
		SFederatedResourceBaseManager: NewFedResourceBaseManager(dt, tableName, keyword, keywordPlural),
	}
}

func (m *SFederatedNamespaceResourceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedNamespaceResourceCreateInput) (*api.FederatedNamespaceResourceCreateInput, error) {
	rInput, err := m.SFederatedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *rInput
	fedNsId := input.FederatednamespaceId
	if fedNsId == "" {
		return nil, httperrors.NewNotEmptyError("federatednamespace_id is empty")
	}
	nsObj, err := GetFedNamespaceManager().GetFedNamespaceByIdOrName(userCred, fedNsId)
	if err != nil {
		return nil, err
	}
	input.FederatednamespaceId = nsObj.GetId()
	input.Federatednamespace = nsObj.GetName()
	return input, nil
}

func (m *SFederatedNamespaceResourceManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.FederatedNamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	q, err := m.SFederatedResourceBaseManager.ListItemFilter(ctx, q, userCred, &input.FederatedResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Federatednamespace != "" {
		ns, err := GetFedNamespaceManager().GetFedNamespaceByIdOrName(userCred, input.Federatednamespace)
		if err != nil {
			return nil, err
		}
		q = q.Equals("federatednamespace_id", ns.GetId())
	}
	return q, nil
}

func (obj *SFederatedNamespaceResource) GetFedNamespace() (*SFederatedNamespace, error) {
	return GetFedNamespaceManager().GetFedNamespace(obj.FederatednamespaceId)
}
