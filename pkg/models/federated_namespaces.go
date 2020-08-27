package models

import (
	"context"

	"k8s.io/kubernetes/pkg/api/legacyscheme"
	"k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/core/validation"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/utils/k8serrors"
)

var (
	fedNamespaceManager *SFederatedNamespaceManager
	_                   IFederatedModelManager = new(SFederatedNamespaceManager)
	_                   IFederatedModel        = new(SFederatedNamespace)
)

func init() {
	GetFedNamespaceManager()
}

func GetFedNamespaceManager() *SFederatedNamespaceManager {
	if fedNamespaceManager == nil {
		fedNamespaceManager = newModelManager(func() db.IModelManager {
			return &SFederatedNamespaceManager{
				SFederatedResourceBaseManager: NewFedResourceBaseManager(
					SFederatedNamespace{},
					"federatednamespaces_tbl",
					"federatednamespace",
					"federatednamespaces",
				),
			}
		}).(*SFederatedNamespaceManager)
	}
	return fedNamespaceManager
}

// +onecloud:swagger-gen-model-singular=federatednamespace
// +onecloud:swagger-gen-model-plural=federatednamespaces
type SFederatedNamespaceManager struct {
	SFederatedResourceBaseManager
}

type SFederatedNamespace struct {
	SFederatedResourceBase
	Spec *api.FederatedNamespaceSpec `list:"user" update:"user" create:"required"`
}

func (m *SFederatedNamespaceManager) GetFedNamespace(id string) (*SFederatedNamespace, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SFederatedNamespace), nil
}

func (m *SFederatedNamespaceManager) GetFedNamespaceByIdOrName(userCred mcclient.IIdentityProvider, id string) (*SFederatedNamespace, error) {
	obj, err := m.FetchByIdOrName(userCred, id)
	if err != nil {
		return nil, err
	}
	return obj.(*SFederatedNamespace), nil
}

func (m *SFederatedNamespaceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.FederatedNamespaceCreateInput) (*api.FederatedNamespaceCreateInput, error) {
	rInput, err := m.SFederatedResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, query, &input.FederatedResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.FederatedResourceCreateInput = *rInput
	nsObj := input.ToNamespace()
	out := new(core.Namespace)
	if err := legacyscheme.Scheme.Convert(nsObj, out, nil); err != nil {
		return nil, errors.Wrap(k8serrors.NewGeneralError(err), "convert to internal namespace object")
	}
	log.Errorf("internal namespace: %#v", out)
	if err := validation.ValidateNamespace(out).ToAggregate(); err != nil {
		return nil, httperrors.NewInputParameterError("%s", err)
	}
	return input, nil
}

func (obj *SFederatedNamespace) GetDetails(base interface{}, isList bool) interface{} {
	out := api.FederatedNamespaceDetails{
		FederatedResourceDetails: obj.SFederatedResourceBase.GetDetails(base, isList).(api.FederatedResourceDetails),
	}
	return out
}

func (obj *SFederatedNamespace) PerformAttachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedNamespaceAttachClusterInput) (*api.FederatedNamespaceAttachClusterInput, error) {
	_, err := obj.GetManager().PerformAttachCluster(obj, ctx, userCred, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (obj *SFederatedNamespace) PerformDetachCluster(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *api.FederatedNamespaceDetachClusterInput) (*api.FederatedNamespaceDetachClusterInput, error) {
	err := obj.GetManager().PerformDetachCluster(obj, ctx, userCred, data.JSON(data))
	if err != nil {
		return nil, err
	}
	return nil, nil
}
