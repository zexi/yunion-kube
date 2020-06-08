package clusters

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models"
)

type SDefaultSystemImportDriver struct {
	*SDefaultImportDriver
}

func NewDefaultSystemImportDriver() *SDefaultSystemImportDriver {
	drv := &SDefaultSystemImportDriver{
		SDefaultImportDriver: NewDefaultImportDriver(),
	}
	drv.providerType = api.ProviderTypeSystem
	drv.clusterResourceType = api.ClusterResourceTypeHost
	return drv
}

func init() {
	models.RegisterClusterDriver(NewDefaultSystemImportDriver())
}

func (d *SDefaultSystemImportDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *SDefaultSystemImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	// test kubeconfig is work
	createData := api.ClusterCreateInput{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	apiServer := createData.ApiServer
	if apiServer == "" {
		return httperrors.NewInputParameterError("ApiServer must provide")
	}
	kubeconfig := createData.Kubeconfig
	cli, _, err := client.BuildClient(apiServer, kubeconfig)
	if err != nil {
		return httperrors.NewNotAcceptableError("Invalid imported kubeconfig: %v", err)
	}
	version, err := cli.Discovery().ServerVersion()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrap(err, "Get kubernetes version"))
	}
	// TODO: inject version info
	log.Infof("Get version: %#v", version)
	return nil
}

func (d *SDefaultSystemImportDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *SDefaultSystemImportDriver) NeedCreateMachines() bool {
	return false
}

func (d *SDefaultSystemImportDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
	return cluster.Kubeconfig, nil
}

func (d *SDefaultSystemImportDriver) ValidateDeleteCondition() error {
	return httperrors.NewNotAcceptableError("system cluster not allow delete")
}
