package model

import (
	"context"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/yunion-kube/pkg/apis"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/clientv2"
)

type RequestContext struct {
	ctx      context.Context
	userCred mcclient.TokenCredential
	cluster  ICluster
	query    *jsonutils.JSONDict
	data     *jsonutils.JSONDict
}

func NewRequestContext(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	cluster ICluster,
	query *jsonutils.JSONDict,
	data *jsonutils.JSONDict,
) *RequestContext {
	return &RequestContext{
		ctx:      ctx,
		userCred: userCred,
		cluster:  cluster,
		query:    query,
		data:     data,
	}
}

func (c *RequestContext) Context() context.Context {
	return c.ctx
}

func (c *RequestContext) Cluster() ICluster {
	return c.cluster
}

func (c *RequestContext) UserCred() mcclient.TokenCredential {
	return c.userCred
}

func (c *RequestContext) ShowAllNamespace() bool {
	return jsonutils.QueryBoolean(c.query, "all_namespace", false)
}

func (c *RequestContext) GetNamespaceByQuery() string {
	namespace, _ := c.query.GetString("namespace")
	return namespace
}

func (c *RequestContext) GetQuery() *jsonutils.JSONDict {
	return c.query
}

type ICluster interface {
	apis.ICluster

	GetHandler() client.ResourceHandler
	GetClient() *clientv2.Client
}

type K8SModelHandler struct {
	modelManager IK8SModelManager
}

func NewK8SModelHandler(manager IK8SModelManager) *K8SModelHandler {
	return &K8SModelHandler{modelManager: manager}
}

func (h *K8SModelHandler) Keyword() string {
	return h.modelManager.Keyword()
}

func (h *K8SModelHandler) KeywordPlural() string {
	return h.modelManager.KeywordPlural()
}

func (h *K8SModelHandler) Filter(f appsrv.FilterHandler) appsrv.FilterHandler {
	if consts.IsRbacEnabled() {
		return auth.AuthenticateWithDelayDecision(f, true)
	}
	return auth.Authenticate(f)
}

func (h *K8SModelHandler) List(ctx *RequestContext, query *jsonutils.JSONDict) (*modulebase.ListResult, error) {
	return ListK8SModels(ctx, h.modelManager, query)
}

func ListK8SModels(ctx *RequestContext, man IK8SModelManager, query *jsonutils.JSONDict) (*modulebase.ListResult, error) {
	var err error
	//var maxLimit int64 = consts.GetMaxPagingLimit()
	baseInput := new(apis.ListInputK8SBase)
	if err := query.Unmarshal(baseInput); err != nil {
		return nil, err
	}
	limit := baseInput.Limit
	if limit == 0 {
		limit = consts.GetDefaultPagingLimit()
	}
	offset := baseInput.Offset
	// paginMarker := baseInput.PagingMarker

	q := man.GetQuery(ctx.Cluster()).Offset(offset).Limit(limit)
	q, err = ListItemFilter(ctx, man, q, query)
	if err != nil {
		return nil, err
	}
	/*filters := jsonutils.GetQueryStringArray(query, "filter")
	if len(filters) > 0 {
		q, err = applyListItemsGeneralFilters(ctx, manager, q, filters)
		if err != nil {
			return nil, err
		}
	}*/
	listResult, err := Query2List(ctx, man, q)
	if err != nil {
		return nil, err
	}
	return calculateListResult(listResult, q.GetTotal(), q.GetLimit(), q.GetOffset()), nil
}

func calculateListResult(data []jsonutils.JSONObject, total, limit, offset int64) *modulebase.ListResult {
	ret := modulebase.ListResult{Data: data, Total: int(total), Limit: int(limit), Offset: int(offset)}
	return &ret
}

func Query2List(ctx *RequestContext, man IK8SModelManager, q IQuery) ([]jsonutils.JSONObject, error) {
	objs, err := q.FetchObjects()
	if err != nil {
		return nil, err
	}
	results := make([]jsonutils.JSONObject, len(objs))
	for i := range objs {
		jsonDict, err := GetObject(ctx, objs[i])
		if err != nil {
			return nil, err
		}
		results[i] = jsonDict
	}
	return results, nil
}

func (h *K8SModelHandler) Get(ctx *RequestContext, id string, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
	namespace := ctx.GetNamespaceByQuery()
	model, err := fetchK8SModel(ctx, h.modelManager, namespace, id, query)
	if err != nil {
		return nil, err
	}

	/*if consts.IsRbacEnabled() {
		if err := db.IsObjectRbacAllowed(model, userCred, policy.PolicyActionGet); err != nil {
			return nil, err
		}
	} else if !model.AllowGetDetails(ctx, userCred, query) {
		return nil, httperrors.NewForbiddenError("Not allow to get details")
	}*/
	return getModelItemDetails(ctx, h.modelManager, model)
}

func getModelItemDetails(
	ctx *RequestContext,
	manager IK8SModelManager, item IK8SModel) (jsonutils.JSONObject, error) {
	return GetDetails(ctx, item)
}

func fetchK8SModel(
	ctx *RequestContext,
	man IK8SModelManager,
	namespace string,
	id string,
	query *jsonutils.JSONDict,
) (IK8SModel, error) {
	cluster := ctx.Cluster()
	cli := cluster.GetHandler()
	resInfo := man.GetK8SResourceInfo()
	obj, err := cli.Get(resInfo.ResourceName, namespace, id)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s %s/%s", resInfo.ResourceName, namespace, id)
	}
	/*uobj := robj.(*unstructured.Unstructured)
	obj := resInfo.Object
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(uobj.UnstructuredContent(), obj); err != nil {
		return nil, errors.Wrap(err, "unstructured marshal")
	}*/
	model, err := NewK8SModelObject(man, cluster, obj)
	if err != nil {
		return nil, err
	}
	return model, nil
}

func NewK8SModelObject(man IK8SModelManager, cluster ICluster, obj runtime.Object) (IK8SModel, error) {
	m, ok := reflect.New(man.Factory().DataType()).Interface().(IK8SModel)
	if !ok {
		return nil, db.ErrInconsistentDataType
	}
	m.SetModelManager(man, m).SetCluster(cluster).SetK8SObject(obj)
	return m, nil
}
