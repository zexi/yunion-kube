package common

import (
	"context"
	"fmt"
	"reflect"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Request struct {
	UserCred mcclient.TokenCredential
	Query    *jsonutils.JSONDict
	Context  context.Context
}

func (r *Request) AllowListItems() bool {
	allNamespace := jsonutils.QueryBoolean(r.Query, "all_namespace", false)
	if allNamespace && !r.UserCred.IsSystemAdmin() {
		return false
	}
	return true
}

func (r *Request) ShowAllNamespace() bool {
	return jsonutils.QueryBoolean(r.Query, "all_namespace", false)
}

func (r *Request) GetNamespace() *NamespaceQuery {
	if r.ShowAllNamespace() {
		return NewNamespaceQuery()
	}
	namespace, _ := r.Query.GetString("namespace")
	if len(namespace) == 0 {
		namespace = r.UserCred.GetProjectName()
	}
	return NewNamespaceQuery(namespace)
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
