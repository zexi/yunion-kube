package clusters

import (
	"context"
	"fmt"
	"strings"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	providerv1 "yunion.io/x/cluster-api-provider-onecloud/pkg/apis/onecloudprovider/v1alpha1"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/k8s"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/manager"
	"yunion.io/x/yunion-kube/pkg/models/types"
	//"yunion.io/x/yunion-kube/pkg/models/clusters/drivers"
)

var ClusterManager *SClusterManager

func init() {
	ClusterManager = &SClusterManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SCluster{}, "kubeclusters_tbl", "kubecluster", "kubeclusters"),
	}
	manager.RegisterClusterManager(ClusterManager)
}

type SClusterManager struct {
	db.SVirtualResourceBaseManager
}

type SCluster struct {
	db.SVirtualResourceBase
	ClusterType   string `nullable:"false" create:"required" list:"user"`
	CloudType     string `nullable:"false" create:"required" list:"user"`
	Mode          string `nullable:"false" create:"required" list:"user"`
	Provider      string `nullable:"false" create:"required" list:"user"`
	ServiceCidr   string `nullable:"false" create:"required" list:"user"`
	ServiceDomain string `nullable:"false" create:"required" list:"user"`
	PodCidr       string `nullable:"true" create:"optional" list:"user"`
	Version       string `nullable:"true" create:"optional" list:"user"`
	Namespace     string `nullable:"true" create:"optional" list:"user"`
	Vip           string `nullable:"true" create:"optional" list:"user"`
}

func SetJSONDataDefault(data *jsonutils.JSONDict, key string, defVal string) string {
	val, _ := data.GetString(key)
	if len(val) == 0 {
		val = defVal
		data.Set(key, jsonutils.NewString(val))
	}
	return val
}

func (m *SClusterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	var (
		clusterType  string
		cloudType    string
		modeType     string
		providerType string
	)

	clusterType = SetJSONDataDefault(data, "cluster_type", string(types.ClusterTypeDefault))
	if !utils.IsInStringArray(clusterType, []string{string(types.ClusterTypeDefault)}) {
		return nil, httperrors.NewInputParameterError("Invalid cluster type: %q", clusterType)
	}

	cloudType = SetJSONDataDefault(data, "cloud_type", string(types.CloudTypePrivate))
	if !utils.IsInStringArray(cloudType, []string{string(types.CloudTypePrivate)}) {
		return nil, httperrors.NewInputParameterError("Invalid cloud type: %q", cloudType)
	}

	modeType = SetJSONDataDefault(data, "mode", string(types.ModeTypeSelfBuild))
	if !utils.IsInStringArray(modeType, []string{string(types.ModeTypeSelfBuild)}) {
		return nil, httperrors.NewInputParameterError("Invalid mode type: %q", modeType)
	}

	providerType = SetJSONDataDefault(data, "provider", string(types.ProviderTypeOnecloud))
	if !utils.IsInStringArray(providerType, []string{string(types.ProviderTypeOnecloud)}) {
		return nil, httperrors.NewInputParameterError("Invalid provider type: %q", providerType)
	}

	serviceCidr := SetJSONDataDefault(data, "service_cidr", types.DefaultServiceCIDR)
	if _, err := netutils.NewIPV4Prefix(serviceCidr); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid service CIDR: %q", serviceCidr)
	}

	serviceDomain := SetJSONDataDefault(data, "service_domain", types.DefaultServiceDomain)
	if len(serviceDomain) == 0 {
		return nil, httperrors.NewInputParameterError("service domain must provided")
	}

	//if err := ValidateCreateData
	res := clusterCreateResource{}
	if err := data.Unmarshal(&res); err != nil {
		return nil, httperrors.NewInputParameterError("Unmarshal: %v", err)
	}
	// TODO: support namespace by userCred
	res.Namespace = res.Name

	if vip := jsonutils.GetAnyString(data, []string{"vip"}); vip != "" {
		if _, err := netutils.NewIPV4Addr(vip); err != nil {
			return nil, httperrors.NewInputParameterError("Invalid vip: %s", vip)
		}
		res.Vip = vip
	}

	if err := m.createResource(&res); err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

type clusterCreateResource struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	ServiceCIDR   string `json:"service_cidr"`
	ServiceDomain string `json:"service_domain"`
	PodCIDR       string `json:"pod_cidr"`
	Vip           string `json:"vip"`
}

func (m *SClusterManager) EnsureNamespace(namespaceName string) error {
	cli, err := m.GetGlobalK8sClient()
	if err != nil {
		return fmt.Errorf("Creating core clientset: %v", err)
	}
	namespace := apiv1.Namespace{
		ObjectMeta: v1.ObjectMeta{
			Name: namespaceName,
		},
	}
	_, err = cli.CoreV1().Namespaces().Create(&namespace)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (m *SClusterManager) DeleteNamespace(namespaceName string) error {
	if namespaceName == apiv1.NamespaceDefault {
		return nil
	}

	cli, err := m.GetGlobalK8sClient()
	if err != nil {
		return fmt.Errorf("Creating core clientset: %v", err)
	}
	err = cli.CoreV1().Namespaces().Delete(namespaceName, &v1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (m *SClusterManager) createResource(data *clusterCreateResource) error {
	cli, err := ClusterManager.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	namespace := data.Namespace
	if err := m.EnsureNamespace(namespace); err != nil {
		return err
	}

	clusterSpec := &providerv1.OneCloudClusterProviderSpec{}
	if data.Vip != "" {
		clusterSpec.NetworkSpec = providerv1.NetworkSpec{
			StaticLB: &providerv1.StaticLB{IPAddress: data.Vip},
		}
	}

	providerValue, err := providerv1.EncodeClusterSpec(clusterSpec)
	if err != nil {
		return err
	}

	cluster := &clusterv1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name: data.Name,
		},
		Spec: clusterv1.ClusterSpec{
			ClusterNetwork: clusterv1.ClusterNetworkingConfig{
				Services:      clusterv1.NetworkRanges{[]string{data.ServiceCIDR}},
				Pods:          clusterv1.NetworkRanges{[]string{data.PodCIDR}},
				ServiceDomain: data.ServiceDomain,
			},
			ProviderSpec: clusterv1.ProviderSpec{
				Value: providerValue,
			},
		},
	}
	if _, err := cli.ClusterV1alpha1().Clusters(namespace).Create(cluster); err != nil {
		return err
	}
	return nil
}

func (m *SClusterManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	clusters := make([]db.IStandaloneModel, len(items))
	for i := range items {
		clusters[i] = items[i]
	}
	models.RunBatchTask(ctx, clusters, userCred, data, "ClusterBatchCreateTask", "")
}

func (m *SClusterManager) FetchClusterByIdOrName(userCred mcclient.TokenCredential, id string) (*SCluster, error) {
	cluster, err := m.FetchByIdOrName(userCred, id)
	if err != nil {
		return nil, err
	}
	return cluster.(*SCluster), nil
}

func (m *SClusterManager) GetGlobalClientConfig() (*rest.Config, error) {
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(nil, "default")
	if err != nil {
		return nil, err
	}
	return cluster.GetK8sRestConfig()
}

func (m *SClusterManager) GetGlobalK8sClient() (*kubernetes.Clientset, error) {
	config, err := m.GetGlobalClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (m *SClusterManager) GetGlobalClient() (*clientset.Clientset, error) {
	conf, err := m.GetGlobalClientConfig()
	if err != nil {
		return nil, err
	}
	return clientset.NewForConfig(conf)
}

func (m *SClusterManager) GetCluster(id string) (*SCluster, error) {
	obj, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return obj.(*SCluster), nil
}

func (m *SCluster) ValidateAddMachine(machine *types.Machine) error {
	cli, err := ClusterManager.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	if _, err := cli.ClusterV1alpha1().Machines(m.Name).Get(machine.Name, v1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return httperrors.NewDuplicateResourceError("Machine %s", m.Name)
}

func (c *SCluster) GetNamespace() string {
	if c.Namespace == "" {
		return c.Name
	}
	return c.Namespace
}

func (c *SCluster) ValidateDeleteCondition(ctx context.Context) error {
	machines, err := manager.MachineManager().GetMachines(c.Id)
	if err != nil {
		return err
	}
	if len(machines) > 0 {
		return httperrors.NewNotEmptyError("Not an empty cluster")
	}
	return nil
}

func (c *SCluster) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("Cluster delete do nothing")
	return nil
}

func (c *SCluster) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return c.SVirtualResourceBase.Delete(ctx, userCred)
}

func (c *SCluster) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	//c.SetStatus(userCred, CLUSTER_STATUS_DELETING, "")
	return c.startRemoveCluster(ctx, userCred)
}

func (c *SCluster) startRemoveCluster(ctx context.Context, userCred mcclient.TokenCredential) error {
	cli, err := ClusterManager.GetGlobalClient()
	if err != nil {
		return httperrors.NewInternalServerError("Get global kubernetes cluster client: %v", err)
	}
	if err := cli.ClusterV1alpha1().Clusters(c.GetNamespace()).Delete(c.Name, &v1.DeleteOptions{}); err != nil {
		if !errors.IsNotFound(err) || strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	return nil
}

func (c *SCluster) allowPerformAction(userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.IsOwner(userCred)
}

func (c *SCluster) AllowPerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func (c *SCluster) PerformTerminate(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.ValidateDeleteCondition(ctx); err != nil {
		return nil, err
	}
	return nil, c.RealDelete(ctx, userCred)
}

func (c *SCluster) AllowGetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, nil)
}

func (c *SCluster) GetControlplaneMachine() (manager.IMachine, error) {
	machines, err := manager.MachineManager().GetMachines(c.Id)
	if err != nil {
		return nil, err
	}
	for _, m := range machines {
		if m.IsControlplane() && m.IsRunning() {
			return m, nil
		}
	}
	return nil, fmt.Errorf("Not found a ready controlplane machine")
}

func (c *SCluster) GetKubeConfig() (string, error) {
	masterMachine, err := c.GetControlplaneMachine()
	if err != nil {
		return "", httperrors.NewInternalServerError("Generate kubeconfig err: %v", err)
	}
	return masterMachine.GetKubeConfig()
}

func (c *SCluster) GetDetailsKubeconfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	conf, err := c.GetKubeConfig()
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewString(conf), "kubeconfig")
	return ret, nil
}

func (c *SCluster) GetAdminKubeconfig() (string, error) {
	return c.GetKubeConfig()
}

func (c *SCluster) GetK8sRestConfig() (*rest.Config, error) {
	kubeconfig, err := c.GetAdminKubeconfig()
	if err != nil {
		return nil, err
	}
	return k8s.GetK8sClientConfig([]byte(kubeconfig))
}

func (c *SCluster) GetK8sClient() (*kubernetes.Clientset, error) {
	config, err := c.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func (c *SCluster) AllowPerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) bool {
	return c.allowPerformAction(userCred, query, data)
}

func (c *SCluster) PerformApplyAddons(ctx context.Context, userCred mcclient.TokenCredential, query, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if err := c.StartApplyAddonsTask(ctx, userCred, data.(*jsonutils.JSONDict), ""); err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *SCluster) StartApplyAddonsTask(ctx context.Context, userCred mcclient.TokenCredential, data *jsonutils.JSONDict, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ClusterApplyAddonsTask", c, userCred, data, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

//func RunBatchTask(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)
