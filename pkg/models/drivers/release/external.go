package release

import (
	"context"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models"
)

func init() {
	models.ReleaseManager.RegisterDriver(newExternalDriver())
}

func newExternalDriver() models.IReleaseDriver {
	return new(externalDriver)
}

type externalDriver struct{}

func (d *externalDriver) GetType() api.RepoType {
	return api.RepoTypeExternal
}

func (d *externalDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, data *api.ReleaseCreateInput) (*api.ReleaseCreateInput, error) {
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(userCred, data.Cluster)
	if err != nil {
		return nil, err
	}
	data.Cluster = cluster.GetId()
	_, err = client.GetManagerByCluster(cluster)
	if err != nil {
		return nil, err
	}
	if data.Namespace == "" {
		return nil, httperrors.NewNotEmptyError("namespace")
	}
	nInput, err := models.ReleaseManager.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerCred, nil, &data.NamespaceResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.NamespaceResourceCreateInput = *nInput
	return data, nil
}

func (d *externalDriver) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerCred mcclient.IIdentityProvider, release *models.SRelease, data *api.ReleaseCreateInput) error {
	release.ClusterId = data.Cluster
	return nil
}
