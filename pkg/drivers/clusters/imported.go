package clusters

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/drivers"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/k8serrors"
)

func init() {
	importDriver := NewDefaultImportDriver()
	importDriver.drivers = drivers.NewDriverManager("")
	importDriver.registerDriver(
		newImportK8sDriver(),
		newImportOpenshiftDriver(),
	)
	models.RegisterClusterDriver(importDriver)
}

type iImportDriver interface {
	GetDistribution() string
	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.ClusterCreateInput, config *rest.Config) error
}

type SDefaultImportDriver struct {
	*SBaseDriver
	drivers *drivers.DriverManager
}

func NewDefaultImportDriver() *SDefaultImportDriver {
	return &SDefaultImportDriver{
		SBaseDriver: newBaseDriver(api.ModeTypeImport, api.ProviderTypeExternal, api.ClusterResourceTypeUnknown),
	}
}

func (d *SDefaultImportDriver) registerDriver(drvs ...iImportDriver) {
	for _, drv := range drvs {
		d.drivers.Register(drv, drv.GetDistribution())
	}
}

func (d *SDefaultImportDriver) getDriver(distro string) iImportDriver {
	drv, err := d.drivers.Get(distro)
	if err != nil {
		panic(fmt.Errorf("Get driver %s: %v", distro, err))
	}
	return drv.(iImportDriver)
}

func (d *SDefaultImportDriver) getRegisterDistros() sets.String {
	ret := make([]string, 0)
	d.drivers.Range(func(key, val interface{}) bool {
		ret = append(ret, key.(string))
		return true
	})
	return sets.NewString(ret...)
}

func (d *SDefaultImportDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *SDefaultImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	// test kubeconfig is work
	createData := new(api.ClusterCreateInput)
	if err := data.Unmarshal(createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
	}
	if createData.Distribution == "" {
		createData.Distribution = api.ImportClusterDistributionK8s
	}
	if !d.getRegisterDistros().Has(createData.Distribution) {
		return httperrors.NewNotSupportedError("Not support import distribution %s", createData.Distribution)
	}
	apiServer := createData.ApiServer
	kubeconfig := createData.Kubeconfig
	restConfig, rawConfig, err := client.BuildClientConfig(apiServer, kubeconfig)
	if err != nil {
		return httperrors.NewNotAcceptableError("Invalid imported kubeconfig: %v", err)
	}
	newKubeconfig, err := runtime.Encode(clientcmdlatest.Codec, rawConfig)
	if err != nil {
		return httperrors.NewNotAcceptableError("Load kubeconfig error: %v", err)
	}
	createData.Kubeconfig = string(newKubeconfig)
	createData.ApiServer = restConfig.Host
	cli, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return k8serrors.NewGeneralError(err)
	}
	version, err := cli.Discovery().ServerVersion()
	if err != nil {
		return k8serrors.NewGeneralError(err)
	}
	// check system cluster duplicate imported
	sysCluster, err := models.ClusterManager.GetSystemCluster()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrap(err, "Get system cluster %v"))
	}
	if sysCluster == nil {
		return httperrors.NewNotFoundError("Not found system cluster %v", sysCluster)
	}
	k8sSvc, err := cli.CoreV1().Services(metav1.NamespaceDefault).Get("kubernetes", metav1.GetOptions{})
	if err != nil {
		return err
	}
	sysCli, err := sysCluster.GetK8sClient()
	if err != nil {
		return httperrors.NewGeneralError(errors.Wrap(err, "Get system cluster k8s client"))
	}
	sysK8SSvc, err := sysCli.CoreV1().Services(metav1.NamespaceDefault).Get("kubernetes", metav1.GetOptions{})
	if err != nil {
		return err
	}
	if k8sSvc.UID == sysK8SSvc.UID {
		return httperrors.NewNotAcceptableError("cluster already imported as default system cluster")
	}

	drv := d.getDriver(createData.Distribution)
	if err := drv.ValidateCreateData(ctx, userCred, ownerId, createData, restConfig); err != nil {
		return errors.Wrapf(err, "check distribution %s", createData.Distribution)
	}

	// TODO: inject version info
	data.Update(jsonutils.Marshal(createData))
	log.Infof("Get version: %#v", version)
	return nil
}

func (d *SDefaultImportDriver) NeedGenerateCertificate() bool {
	return false
}

func (d *SDefaultImportDriver) NeedCreateMachines() bool {
	return false
}

func (d *SDefaultImportDriver) GetKubeconfig(cluster *models.SCluster) (string, error) {
	return cluster.Kubeconfig, nil
}

func (d *SDefaultImportDriver) ValidateCreateMachines(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	c *models.SCluster,
	imageRepo *api.ImageRepository,
	data []*api.CreateMachineData) error {
	return httperrors.NewBadRequestError("Not support add machines")
}
