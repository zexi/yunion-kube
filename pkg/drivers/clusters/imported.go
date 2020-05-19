package clusters

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/utils/k8serrors"
)

type SDefaultImportDriver struct {
	*SBaseDriver
}

func NewDefaultImportDriver() *SDefaultImportDriver {
	return &SDefaultImportDriver{
		SBaseDriver: newBaseDriver(apis.ModeTypeImport, apis.ProviderTypeExternal, apis.ClusterResourceTypeUnknown),
	}
}

func init() {
	models.RegisterClusterDriver(NewDefaultImportDriver())
}

func (d *SDefaultImportDriver) GetK8sVersions() []string {
	return []string{}
}

func (d *SDefaultImportDriver) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) error {
	// test kubeconfig is work
	createData := apis.ClusterCreateInput{}
	if err := data.Unmarshal(&createData); err != nil {
		return httperrors.NewInputParameterError("Unmarshal to CreateClusterData: %v", err)
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
	imageRepo *apis.ImageRepository,
	data []*apis.CreateMachineData) error {
	return httperrors.NewBadRequestError("Not support add machines")
}
