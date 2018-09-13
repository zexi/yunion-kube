package k8s

import (
	"context"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/yunion-kube/pkg/models"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/errors"
)

type IK8sResourceHandler interface {
	dispatcher.IMiddlewareFilter

	Keyword() string
	KeywordPlural() string

	List(ctx context.Context, query *jsonutils.JSONDict) (common.ListResource, error)

	Get(ctx context.Context, id string, query *jsonutils.JSONDict) (interface{}, error)

	Create(ctx context.Context, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (interface{}, error)

	Update(ctx context.Context, id string, query *jsonutils.JSONDict, data *jsonutils.JSONDict) (interface{}, error)

	Delete(ctx context.Context, id string, query *jsonutils.JSONDict, data *jsonutils.JSONDict) error
}

type IK8sResourceManager interface {
	Keyword() string
	KeywordPlural() string

	InNamespace() bool

	// list hooks
	AllowListItems(req *common.Request) bool
	List(req *common.Request) (common.ListResource, error)

	// get hooks
	AllowGetItem(req *common.Request, id string) bool
	Get(req *common.Request, id string) (interface{}, error)

	// create hooks
	AllowCreateItem(req *common.Request) bool
	ValidateCreateData(req *common.Request) error
	Create(req *common.Request) (interface{}, error)

	// update hooks
	AllowUpdateItem(req *common.Request, id string) bool
	Update(req *common.Request, id string) (interface{}, error)

	// delete hooks
	AllowDeleteItem(req *common.Request, id string) bool
	Delete(req *common.Request, id string) error

	IsRawResource() bool
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

func getCluster(query, data *jsonutils.JSONDict, userCred mcclient.TokenCredential) (*models.SCluster, error) {
	var clusterId string
	for _, src := range []*jsonutils.JSONDict{query, data} {
		clusterId, _ = src.GetString("cluster")
		if clusterId != "" {
			break
		}
	}
	if clusterId == "" {
		clusterId = "default"
	}
	cluster, err := models.ClusterManager.FetchClusterByIdOrName(userCred.GetProjectId(), clusterId)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func getK8sClient(query, data *jsonutils.JSONDict, userCred mcclient.TokenCredential) (kubernetes.Interface, *rest.Config, error) {
	cluster, err := getCluster(query, data, userCred)
	if err != nil {
		return nil, nil, err
	}
	config, err := cluster.GetK8sRestConfig()
	if err != nil {
		return nil, nil, err
	}
	cli, err := kubernetes.NewForConfig(config)
	return cli, config, err
}

func NewCloudK8sRequest(ctx context.Context, query, data *jsonutils.JSONDict) (*common.Request, error) {
	userCred := getUserCredential(ctx)
	k8sCli, config, err := getK8sClient(query, data, userCred)
	if err != nil {
		return nil, err
	}
	req := &common.Request{
		K8sClient: k8sCli,
		K8sConfig: config,
		UserCred:  userCred,
		Query:     query,
		Data:      data,
		Context:   ctx,
	}
	return req, nil
}

func (h *K8sResourceHandler) List(ctx context.Context, query *jsonutils.JSONDict) (common.ListResource, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, errors.NewJSONClientError(err)
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
) (common.ListResource, error) {
	ret, err := man.List(req)
	return ret, err
}

func (h *K8sResourceHandler) Get(ctx context.Context, id string, query *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if !h.resourceManager.AllowGetItem(req, id) {
		return nil, httperrors.NewForbiddenError("Not allow to get item")
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

func (h *K8sResourceHandler) Create(ctx context.Context, query, data *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	if !h.resourceManager.AllowCreateItem(req) {
		return nil, httperrors.NewForbiddenError("Not allow to create item")
	}

	return doCreateItem(h.resourceManager, req)
}

func (h *K8sResourceHandler) Update(ctx context.Context, id string, query, data *jsonutils.JSONDict) (interface{}, error) {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	obj, err := doUpdateItem(h.resourceManager, req, id)

	if err != nil {
		return nil, errors.NewJSONClientError(err)
	}
	return obj, err
}

func doUpdateItem(man IK8sResourceManager, req *common.Request, id string) (interface{}, error) {
	if !man.AllowUpdateItem(req, id) {
		return nil, httperrors.NewForbiddenError("Not allow to delete")
	}
	return man.Update(req, id)
}

func (h *K8sResourceHandler) Delete(ctx context.Context, id string, query, data *jsonutils.JSONDict) error {
	req, err := NewCloudK8sRequest(ctx, query, data)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	if h.resourceManager.IsRawResource() {
		err = doRawDelete(h.resourceManager, req, id)
	} else {
		err = doDeleteItem(h.resourceManager, req, id)
	}

	if err != nil {
		return errors.NewJSONClientError(err)
	}
	return nil
}

func doRawDelete(man IK8sResourceManager, req *common.Request, id string) error {
	verber, err := req.GetVerberClient()
	if err != nil {
		return err
	}

	kind := man.Keyword()
	namespace := ""
	inNamespace := man.InNamespace()
	if inNamespace {
		namespace = req.GetDefaultNamespace()
	}
	if err := verber.Delete(kind, inNamespace, namespace, id); err != nil {
		return err
	}
	return nil
}

func doDeleteItem(man IK8sResourceManager, req *common.Request, id string) error {
	if !man.AllowDeleteItem(req, id) {
		return httperrors.NewForbiddenError("Not allow to delete")
	}
	return man.Delete(req, id)
}
