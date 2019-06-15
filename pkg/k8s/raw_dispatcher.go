package k8s

import (
	"context"
	"fmt"
	"net/http"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	clientapi "yunion.io/x/yunion-kube/pkg/k8s/client/api"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/errors"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

func AddRawResourceDispatcher(prefix string, app *appsrv.Application) {
	log.Infof("Register k8s raw resource dispatcher")
	clusterPrefix := getClusterPrefix(prefix)

	rawResourcePrefix := fmt.Sprintf("%s/_raw/<kind>/<name>", clusterPrefix)

	// GET raw resource
	app.AddHandler("GET", rawResourcePrefix, auth.Authenticate(getResourceHandler))

	// Get raw resource yaml
	app.AddHandler("GET", fmt.Sprintf("%s/yaml", rawResourcePrefix), auth.Authenticate(getResourceYAMLHandler))

	// PUT raw resource
	app.AddHandler("PUT", rawResourcePrefix, auth.Authenticate(putResourceHandler))

	// DELETE raw resource
	app.AddHandler("DELETE", rawResourcePrefix, auth.Authenticate(deleteResourceHandler))
}

func NewCommonRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) (*common.Request, error) {
	_, query, body := _fetchEnv(ctx, w, r)
	queryDict := jsonutils.NewDict()
	dataDict := jsonutils.NewDict()
	if query != nil {
		queryDict = query.(*jsonutils.JSONDict)
	}
	if body != nil {
		dataDict = body.(*jsonutils.JSONDict)
	}
	return NewCloudK8sRequest(ctx, queryDict, dataDict)
}

type verberEnv struct {
	client      clientapi.ResourceVerber
	kind        string
	namespace   string
	inNamespace bool
	name        string
	request     *common.Request
}

func fetchVerberEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*verberEnv, error) {
	req, err := NewCommonRequest(ctx, w, r)
	if err != nil {
		return nil, err
	}
	cli, err := req.GetVerberClient()
	if err != nil {
		return nil, err
	}
	params := req.GetParams()
	kindPlural := params["<kind>"]
	name := params["<name>"]
	kind := api.TrimKindPlural(kindPlural)
	resourceSpec, ok := api.KindToAPIMapping[kind]
	if !ok {
		return nil, fmt.Errorf("Not found %q resource kind spec", kind)
	}
	inNamespace := resourceSpec.Namespaced
	namespace := ""
	if inNamespace {
		namespace = req.GetDefaultNamespace()
	}
	env := &verberEnv{
		client:      cli,
		kind:        kind,
		inNamespace: inNamespace,
		namespace:   namespace,
		name:        name,
		request:     req,
	}
	return env, nil
}

func (env *verberEnv) Get() (runtime.Object, error) {
	return env.client.Get(env.kind, env.inNamespace, env.namespace, env.name)
}

func (env *verberEnv) Put() error {
	rawStr, err := env.request.Data.GetString()
	if err != nil {
		return httperrors.NewInputParameterError("Get body string error: %v", err)
	}
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(rawStr), nil, nil)
	if err != nil {
		return httperrors.NewInputParameterError("Decode error: %v", err)
	}
	log.Debugf("Input %s, get object: %#v", rawStr, obj)
	//putSpec := runtime.Unknown{}
	//err = json.NewDecoder(strings.NewReader(rawStr)).Decode(&putSpec)
	//if err != nil {
	//return err
	//}
	return env.client.Put(env.kind, env.inNamespace, env.namespace, env.name, obj)
}

func (env *verberEnv) Delete() error {
	return env.client.Delete(env.kind, env.inNamespace, env.namespace, env.name)
}

func getResourceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	obj, err := env.Get()
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	SendJSON(w, obj)
}

func getResourceYAMLHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	obj, err := env.Get()
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	SendYAML(w, obj)
}

func putResourceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	err = env.Put()
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func deleteResourceHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	env, err := fetchVerberEnv(ctx, w, r)
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	err = env.Delete()
	if err != nil {
		errors.GeneralServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
