package common

import (
	"context"
	"fmt"

	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/mcclient"
	clientapi "yunion.io/x/yunion-kube/pkg/k8s/client/api"

	helmclient "yunion.io/x/yunion-kube/pkg/helm/client"
	k8sclient "yunion.io/x/yunion-kube/pkg/k8s/client"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Request struct {
	K8sClient client.Interface
	K8sConfig *rest.Config
	UserCred  mcclient.TokenCredential
	Query     *jsonutils.JSONDict
	Data      *jsonutils.JSONDict
	Context   context.Context
}

func (r *Request) AllowListItems() bool {
	allNamespace := jsonutils.QueryBoolean(r.Query, "all_namespace", false)
	if allNamespace && !r.UserCred.IsSystemAdmin() {
		return false
	}
	return true
}

func (r *Request) AllowCreateItem() bool {
	if r.UserCred.IsSystemAdmin() {
		return true
	}
	ns, err := r.GetNamespaceByData()
	if err != nil {
		log.Errorf("Get namespace from data error: %v", err)
		return false
	}
	// TODO: support isOwner check
	return ns == r.UserCred.GetProjectName()
}

func (r *Request) ShowAllNamespace() bool {
	return jsonutils.QueryBoolean(r.Query, "all_namespace", false)
}

func (r *Request) GetNamespaceQuery() *NamespaceQuery {
	if r.ShowAllNamespace() {
		return NewNamespaceQuery()
	}
	namespace, _ := r.Query.GetString("namespace")
	if len(namespace) == 0 {
		namespace = r.UserCred.GetProjectName()
	}
	return NewNamespaceQuery(namespace)
}

func (r *Request) GetK8sClient() client.Interface {
	return r.K8sClient
}

func (r *Request) GetVerberClient() (clientapi.ResourceVerber, error) {
	cli := r.GetK8sClient()
	return k8sclient.NewResourceVerber(
		cli.CoreV1().RESTClient(),
		cli.ExtensionsV1beta1().RESTClient(),
		cli.AppsV1beta2().RESTClient(),
		cli.BatchV1().RESTClient(),
		cli.BatchV1beta1().RESTClient(),
		cli.AutoscalingV1().RESTClient(),
		cli.StorageV1().RESTClient()), nil
}

func (r *Request) GetHelmClient() (*helmclient.HelmTunnelClient, error) {
	k8scli := r.GetK8sClient()
	k8sconfig := r.K8sConfig
	return helmclient.NewHelmTunnelClient(k8scli, k8sconfig)
}

func (r *Request) GetNamespaceByQuery() (string, error) {
	if r.Query == nil {
		return "", fmt.Errorf("query is nil")
	}
	return r.Query.GetString("namespace")
}

func (r *Request) GetNamespaceByData() (string, error) {
	if r.Data == nil {
		return "", fmt.Errorf("data is nil")
	}
	return r.Data.GetString("namespace")
}

func (r *Request) GetDefaultNamespace() string {
	ns, _ := r.GetNamespaceByQuery()
	if ns != "" {
		return ns
	}
	ns, _ = r.GetNamespaceByData()
	if ns == "" {
		ns = r.UserCred.GetProjectName()
	}
	return ns
}

func NewDataSelectQuery(query jsonutils.JSONObject) *dataselect.DataSelectQuery {
	limit, _ := query.Int("limit")
	if limit == 0 {
		limit = 20
	}
	offset, _ := query.Int("offset")
	limitQ := dataselect.NewLimitQuery(int(limit))
	offsetQ := dataselect.NewOffsetQuery(int(offset))

	filterQ := dataselect.NoFilter
	filterRawCond := []string{}
	name, _ := query.GetString("name")
	if name != "" {
		filterRawCond = append(filterRawCond, dataselect.NameProperty, name)
	}
	if len(filterRawCond) != 0 {
		filterQ = dataselect.NewFilterQuery(filterRawCond)
	}
	return dataselect.NewDataSelectQuery(
		dataselect.NoSort, // TODO
		filterQ,
		limitQ,
		offsetQ,
	)
}

func (r *Request) ToQuery() *dataselect.DataSelectQuery {
	return NewDataSelectQuery(r.Query)
}

func (r *Request) GetParams() map[string]string {
	return appctx.AppContextParams(r.Context)
}

type ListResource interface {
	api.IListMeta

	GetResponseData() interface{}
}

func ListResource2JSONWithKey(list ListResource, key string) map[string]interface{} {
	ret := make(map[string]interface{})
	if list.GetTotal() > 0 {
		ret["total"] = list.GetTotal()
	}
	if list.GetLimit() > 0 {
		ret["limit"] = list.GetLimit()
	}
	if list.GetOffset() > 0 {
		ret["offset"] = list.GetOffset()
	}
	ret[key] = list.GetResponseData()
	return ret
}