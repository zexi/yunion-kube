package k8s

import (
	"context"
	"fmt"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/appctx"
	"github.com/yunionio/pkg/appsrv"
	"github.com/yunionio/pkg/httperrors"

	"github.com/yunionio/mcclient/modules"
)

func AddK8sResourceDispatcher(prefix string, app *appsrv.Application, handler IK8sResourceHandler) {
	log.Infof("Register k8s resource %s", handler.Keyword())

	clusterPrefix := fmt.Sprintf("%s/clusters/<clusterid>", prefix)
	plural := handler.KeywordPlural()
	metadata := map[string]interface{}{"manager": handler}
	tags := map[string]string{"resource": plural}

	// list resources
	app.AddHandler2("GET", fmt.Sprintf("%s/%s", clusterPrefix, plural), handler.Filter(listHandler), metadata, "list", tags)
}

func _fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	// trace := appsrv.AppContextTrace(ctx)
	params := appctx.AppContextParams(ctx)
	query, e := jsonutils.ParseQueryString(r.URL.RawQuery)
	if e != nil {
		log.Errorf("Parse query string %s failed: %s", r.URL.RawQuery, e)
	}
	var body jsonutils.JSONObject = nil
	if r.Method == "PUT" || r.Method == "POST" || r.Method == "DELETE" || r.Method == "PATCH" {
		body, e = appsrv.FetchJSON(r)
		if e != nil {
			log.Errorf("Fail to decode JSON request body: %s", e)
		}
	}
	return params, query, body
}

func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (
	IK8sResourceHandler, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params, query, body := _fetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	handler, ok := metadata["manager"].(IK8sResourceHandler)
	if !ok {
		log.Fatalf("No manager found for URL: %s", r.URL)
	}
	return handler, params, query, body
}

func listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	handler, _, query, _ := fetchEnv(ctx, w, r)
	handleList(ctx, w, handler, query)
}

func handleList(ctx context.Context, w http.ResponseWriter, handler IK8sResourceHandler, query jsonutils.JSONObject) {
	result, err := handler.List(ctx, query.(*jsonutils.JSONDict))
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, modules.ListResult2JSONWithKey(result, handler.KeywordPlural()))
}
