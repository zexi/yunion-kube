package k8s

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/resources/common"
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
	AllowListItems(req *common.Request) bool
	List(k8sCli kubernetes.Interface, req *common.Request) (common.ListResource, error)
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

func getCluster(ctx context.Context, userCred mcclient.TokenCredential) (*models.SCluster, error) {
	params := appctx.AppContextParams(ctx)
	clusterId := params["<clusterid>"]
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(userCred.GetProjectId(), clusterId)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func getK8sClient(ctx context.Context, userCred mcclient.TokenCredential) (kubernetes.Interface, error) {
	cluster, err := getCluster(ctx, userCred)
	if err != nil {
		return nil, err
	}
	config, err := cluster.GetK8sRestConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func getCloudK8sEnv(ctx context.Context, query *jsonutils.JSONDict) (*common.Request, kubernetes.Interface, error) {
	userCred := getUserCredential(ctx)
	k8sCli, err := getK8sClient(ctx, userCred)
	if err != nil {
		return nil, nil, err
	}
	req := &common.Request{
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
	req *common.Request,
) (*modules.ListResult, error) {
	ret, err := man.List(k8sCli, req)
	if err != nil {
		return nil, err
	}
	count := ret.GetTotal()
	if count == 0 {
		emptyList := modules.ListResult{Data: []jsonutils.JSONObject{}}
		return &emptyList, nil
	}
	retResult := modules.ListResult{Data: ret.GetData(), Total: ret.GetTotal(), Limit: ret.GetLimit(), Offset: ret.GetOffset()}
	return &retResult, nil
}
