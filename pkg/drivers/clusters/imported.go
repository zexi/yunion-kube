package clusters

import (
	"context"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/models/types"
)

type SDefaultImportDriver struct {
	*SBaseDriver
}

func NewDefaultImportDriver() *SDefaultImportDriver {
	return &SDefaultImportDriver{
		SBaseDriver: newBaseDriver(),
	}
}

func init() {
	clusters.RegisterClusterDriver(NewDefaultImportDriver())
}

func (d *SDefaultImportDriver) GetMode() types.ModeType {
	return types.ModeTypeImport
}

func (d *SDefaultImportDriver) GetProvider() types.ProviderType {
	return types.ProviderTypeExternal
}

func (d *SDefaultImportDriver) GetResourceType() types.ClusterResourceType {
	return types.ClusterResourceTypeUnknown
}

func (d *SDefaultImportDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *SDefaultImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	// test kubeconfig is work
	createData := types.CreateClusterData{}
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

func (d *SDefaultImportDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *SDefaultImportDriver) NeedCreateMachines() bool {
	return false
}

func (d *SDefaultImportDriver) GetKubeconfig(cluster *clusters.SCluster) (string, error) {
	return cluster.Kubeconfig, nil
}

func (d *SDefaultImportDriver) ValidateCreateMachines(ctx context.Context, userCred mcclient.TokenCredential, c *clusters.SCluster, data []*types.CreateMachineData) error {
	return httperrors.NewBadRequestError("Not support add machines")
}
