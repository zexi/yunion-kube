package common

import (
	"context"
	"fmt"
	"reflect"

	client "k8s.io/client-go/kubernetes"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/mcclient"
	clientapi "yunion.io/x/yunion-kube/pkg/k8s/client/api"

	k8sclient "yunion.io/x/yunion-kube/pkg/k8s/client"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Request struct {
	K8sClient client.Interface
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

func (r *Request) GetNamespaceByQuery() (string, error) {
	return r.Query.GetString("namespace")
}

func (r *Request) GetNamespaceByData() (string, error) {
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

func (r *Request) ToQuery() *dataselect.DataSelectQuery {
	limit, _ := r.Query.Int("limit")
	if limit == 0 {
		limit = 20
	}
	offset, _ := r.Query.Int("offset")
	limitQ := dataselect.NewLimitQuery(int(limit))
	offsetQ := dataselect.NewOffsetQuery(int(offset))
	return dataselect.NewDataSelectQuery(
		dataselect.NoSort,   // TODO
		dataselect.NoFilter, // TODO
		limitQ,
		offsetQ,
	)
}

func (r *Request) GetParams() map[string]string {
	return appctx.AppContextParams(r.Context)
}

type ListResource interface {
	api.IListMeta

	GetResponseData() interface{}
}

func ToListJsonData(data interface{}) ([]jsonutils.JSONObject, error) {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Can't traverse non-slice value, kind: %v", v.Kind())
	}

	ret := make([]jsonutils.JSONObject, 0)
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		funcV := item.MethodByName("ToListItem")
		if !funcV.IsValid() || funcV.IsNil() {
			return nil, fmt.Errorf("Item kind %v not implement ToListItem function", item.Kind())
		}
		out := funcV.Call([]reflect.Value{})
		if len(out) != 1 {
			return nil, fmt.Errorf("Invalid return value: %#v", out)
		}
		ret = append(ret, out[0].Interface().(jsonutils.JSONObject))
	}
	return ret, nil
}
