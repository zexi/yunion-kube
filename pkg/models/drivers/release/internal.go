package release

import (
	"context"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/models"
)

func init() {
	models.ReleaseManager.RegisterDriver(newInternalDriver())
}

func newInternalDriver() models.IReleaseDriver {
	return new(internalDriver)
}

type internalDriver struct{}

func (d *internalDriver) GetType() apis.RepoType {
	return apis.RepoTypeInternal
}

func (d *internalDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, data *apis.ReleaseCreateInput) (*apis.ReleaseCreateInput, error) {
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
	return data, nil
}

func (d *internalDriver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, release *models.SRelease, data *apis.ReleaseCreateInput) error {
	release.ClusterId = data.Cluster
	release.Namespace = ownerCred.GetProjectId()
	cluster, err := release.GetCluster()
	if err != nil {
		return errors.Wrap(err, "get cluster")
	}
	if err := models.EnsureNamespace(cluster, release.Namespace); err != nil {
		return errors.Wrap(err, "ensure namespace")
	}
	return nil
}
