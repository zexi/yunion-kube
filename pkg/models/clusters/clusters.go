package clusters

import (
	"context"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"

	"yunion.io/x/jsonutils"
	//"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/models/types"
	//"yunion.io/x/yunion-kube/pkg/models/clusters/drivers"
)

var ClusterManager *SClusterManager

func init() {
	ClusterManager = &SClusterManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(SCluster{}, "kubeclusters_tbl", "kubecluster", "kubeclusters"),
	}
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

	return m.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data)
}

func (m *SClusterManager) OnCreateComplete(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	clusters := make([]db.IStandaloneModel, len(items))
	for i := range items {
		clusters[i] = items[i]
	}
	models.RunBatchTask(ctx, clusters, userCred, data, "ClusterBatchCreateTask", "")
}

func (m *SClusterManager) GetGlobalClientConfig() (*rest.Config, error) {
	cluster, err := models.ClusterManager.FetchClusterById("default")
	if err != nil {
		return nil, err
	}
	return cluster.GetK8sRestConfig()
}

func (m *SClusterManager) GetGlobalClient() (*clientset.Clientset, error) {
	conf, err := m.GetGlobalClientConfig()
	if err != nil {
		return nil, err
	}
	return clientset.NewForConfig(conf)
}

//func RunBatchTask(ctx context.Context, items []db.IModel, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject)
