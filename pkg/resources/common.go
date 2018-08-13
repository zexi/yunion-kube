package resources

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type Request struct {
	UserCred mcclient.TokenCredential
	Query    *jsonutils.JSONDict
	Context  context.Context
}

func (r *Request) GetNamespace() string {
	return r.UserCred.GetProjectName()
}

type ListResource interface {
	Limit() int
	Total() int
	Offset() int
	Data() []jsonutils.JSONObject
}
