package resources

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
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
