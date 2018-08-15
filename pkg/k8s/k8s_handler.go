package k8s

import (
	"context"
	"fmt"

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

	Get(ctx context.Context, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error)

	Create(ctx context.Context, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)

	//Update(ctx context.Context, id string, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error)

	Delete(ctx context.Context, id string, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (jsonutils.JSONObject, error)
}

type IK8sResourceManager interface {
	Keyword() string
	KeywordPlural() string

	// list hooks
	AllowListItems(req *common.Request) bool
	List(req *common.Request) (common.ListResource, error)

	// get hooks
	Get(req *common.Request, id string) (jsonutils.JSONObject, error)

	// create hooks
	ValidateCreateData(req *common.Request) error
	Create(req *common.Request) (jsonutils.JSONObject, error)

	// delete hooks
	Delete(req *common.Request, id string) (jsonutils.JSONObject, error)
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

func NewCloudK8sRequest(ctx context.Context, query, data *jsonutils.JSONDict) (*common.Request, error) {
	userCred := getUserCredential(ctx)
	k8sCli, err := getK8sClient(ctx, userCred)
	if err != nil {
		return nil, err
	}
	req := &common.Request{
		K8sClient: k8sCli,
		UserCred:  userCred,
		Query:     query,
		Data:      data,
		Context:   ctx,
	}
	return req, nil
}

func (h *K8sResourceHandler) List(ctx context.Context, query *jsonutils.JSONDict) (*modules.ListResult, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if !h.resourceManager.AllowListItems(req) {
		return nil, httperrors.NewForbiddenError("Not allow to list")
	}
	items, err := listItems(h.resourceManager, req)
	if err != nil {
		log.Errorf("Fail to list items: %v", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return items, nil
}

func listItems(
	man IK8sResourceManager,
	req *common.Request,
) (*modules.ListResult, error) {
	ret, err := man.List(req)
	if err != nil {
		return nil, err
	}
	count := ret.GetTotal()
	if count == 0 {
		emptyList := modules.ListResult{Data: []jsonutils.JSONObject{}}
		return &emptyList, nil
	}
	data, err := common.ToListJsonData(ret.GetResponseData())
	if err != nil {
		return nil, err
	}
	retResult := modules.ListResult{Data: data, Total: ret.GetTotal(), Limit: ret.GetLimit(), Offset: ret.GetOffset()}
	return &retResult, nil
}

func (h *K8sResourceHandler) Get(ctx context.Context, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return h.resourceManager.Get(req, id)
}

func doCreateItem(man IK8sResourceManager, req *common.Request) (jsonutils.JSONObject, error) {
	err := man.ValidateCreateData(req)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	res, err := man.Create(req)
	if err != nil {
		log.Errorf("Fail to create resource: %v", err)
		return nil, err
	}
	return jsonutils.Marshal(res), nil
}

func (h *K8sResourceHandler) Create(ctx context.Context, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if !req.AllowCreateItem() {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	return doCreateItem(h.resourceManager, req)
}

func (h *K8sResourceHandler) Delete(ctx context.Context, id string, query, data *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	_, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	//if !h.resourceManager.AllowDeleteItem() {
	//return nil, httperrors.NewForbiddenError("%s not allow to delete", manager.KeywordPlural())
	//}
	return nil, fmt.Errorf("not impl")
}
