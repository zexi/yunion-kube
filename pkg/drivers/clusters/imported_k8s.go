package clusters

import (
	"context"

	"k8s.io/client-go/rest"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

type importK8sDriver struct{}

func newImportK8sDriver() iImportDriver {
	return new(importK8sDriver)
}

func (d *importK8sDriver) GetDistribution() string {
	return api.ImportClusterDistributionK8s
}

func (d *importK8sDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error {
	return nil
}
