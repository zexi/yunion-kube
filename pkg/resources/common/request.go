package common

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"

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

type ListResource interface {
	api.IListMeta

	GetData() []jsonutils.JSONObject
}
