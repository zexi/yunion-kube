package k8s

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/auth"
	"github.com/yunionio/mcclient/modules"
	"github.com/yunionio/pkg/appctx"
	"github.com/yunionio/pkg/appsrv"
	"github.com/yunionio/pkg/appsrv/dispatcher"
	"github.com/yunionio/pkg/httperrors"
	"k8s.io/client-go/kubernetes"

	"yunion.io/yunion-kube/pkg/models"
	"yunion.io/yunion-kube/pkg/resources"
)

type IK8sResourceHandler interface {
	dispatcher.IMiddlewareFilter

	Keyword() string
	KeywordPlural() string

	List(ctx context.Context, query *jsonutils.JSONDict) (*modules.ListResult, error)
}

type IK8sResourceManager interface {
	Keyword() string
	KeywordPlural() string

	// list hooks
	AllowListItems(req *resources.Request) bool
	List(k8sCli kubernetes.Interface, req *resources.Request) (resources.ListResource, error)
}

type K8sResourceHandler struct {
	resourceManager IK8sResourceManager
}

func NewK8sResourceHandler(man IK8sResourceManager) *K8sResourceHandler {
	return &K8sResourceHandler{man}
}

func (h *K8sResourceHandler) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	return auth.Authenticate(f)
}

func (h *K8sResourceHandler) Keyword() string {
	return h.resourceManager.Keyword()
}

func (h *K8sResourceHandler) KeywordPlural() string {
	return h.resourceManager.KeywordPlural()
}

func getUserCredential(ctx context.Context) mcclient.TokenCredential {
	token := auth.FetchUserCredential(ctx)
	if token == nil {
		log.Fatalf("user token credential not found?")
	}
	return token
}

func getCluster(ctx context.Context) (*models.SCluster, error) {
	params := appctx.AppContextParams(ctx)
	clusterId := params["<clusterid>"]
	cluster, err := models.ClusterManager.FetchClusterById(clusterId)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func getK8sClient(ctx context.Context) (kubernetes.Interface, error) {
	cluster, err := getCluster(ctx)
	if err != nil {
		return nil, err
	}
	config, err := cluster.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func getCloudK8sEnv(ctx context.Context, query *jsonutils.JSONDict) (*resources.Request, kubernetes.Interface, error) {
	userCred := getUserCredential(ctx)
	k8sCli, err := getK8sClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	req := &resources.Request{
		UserCred: userCred,
		Query:    query,
		Context:  ctx,
	}
	return req, k8sCli, nil
}

func (h *K8sResourceHandler) List(ctx context.Context, query *jsonutils.JSONDict) (*modules.ListResult, error) {
	req, k8sCli, err := getCloudK8sEnv(ctx, query)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if !h.resourceManager.AllowListItems(req) {
		return nil, httperrors.NewForbiddenError("Not allow to list")
	}
	items, err := listItems(h.resourceManager, k8sCli, req)
	if err != nil {
		log.Errorf("Fail to list items: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return items, nil
}

func listItems(
	man IK8sResourceManager,
	k8sCli kubernetes.Interface,
	req *resources.Request,
) (*modules.ListResult, error) {
	ret, err := man.List(k8sCli, req)
	if err != nil {
		return nil, err
	}
	count := ret.Total()
	if count == 0 {
		emptyList := modules.ListResult{Data: []jsonutils.JSONObject{}}
		return &emptyList, nil
	}
	retResult := modules.ListResult{Data: ret.Data(), Total: ret.Total(), Limit: ret.Limit(), Offset: ret.Offset()}
	return &retResult, nil
}
