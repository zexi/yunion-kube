package release

import (
	"context"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/models"
)

func init() {
	models.ReleaseManager.RegisterDriver(newInternalDriver())
}

func newInternalDriver() models.IReleaseDriver {
	return new(internalDriver)
}

type internalDriver struct{}

func (d *internalDriver) GetType() api.RepoType {
	return api.RepoTypeInternal
}

func (d *internalDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, data *api.ReleaseCreateInput) (*api.ReleaseCreateInput, error) {
	if data.Namespace != "" {
		return nil, httperrors.NewNotAcceptableError("%s release can not specify namespace", d.GetType())
	}
	if data.Cluster != "" {
		return nil, httperrors.NewNotAcceptableError("%s release can not specify cluster", d.GetType())
	}
	data.Namespace = ownerCred.GetProjectId()
	sysCls, err := models.ClusterManager.GetSystemCluster()
	if err != nil {
		return nil, err
	}
	if sysCls == nil {
		return nil, httperrors.NewNotFoundError("system cluster not found")
	}
	data.Cluster = sysCls.GetId()
	nsData := new(api.NamespaceCreateInputV2)
	nsData.Name = ownerCred.GetProjectId()
	nsData.Cluster = sysCls.GetId()
	ns, err := models.NamespaceManager.EnsureNamespace(ctx, userCred, ownerCred, sysCls, nsData)
	if err != nil {
		return nil, errors.Wrap(err, "ensure namespace")
	}
	data.Namespace = ns.GetId()
	return data, nil
}

func (d *internalDriver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, release *models.SRelease, data *api.ReleaseCreateInput) error {
	release.ClusterId = data.Cluster
	release.NamespaceId = data.Namespace
	return nil
}