package clusters

import (
	"context"

	configv1client "github.com/openshift/client-go/config/clientset/versioned/typed/config/v1"
	"k8s.io/client-go/rest"

	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
)

type importOpenshiftDriver struct{}

func newImportOpenshiftDriver() iImportDriver {
	return new(importK8sDriver)
}

func (d *importOpenshiftDriver) GetDistribution() string {
	return api.ImportClusterDistributionK8s
}

func (d *importOpenshiftDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error {
	configv1client.NewForConfig(config)
	return nil
}
