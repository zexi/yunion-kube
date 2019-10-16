package common

import (
	"context"
	"fmt"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/sets"
	clientapi "yunion.io/x/yunion-kube/pkg/k8s/client/api"

	yclient "yunion.io/x/yunion-kube/pkg/client"
	helmclient "yunion.io/x/yunion-kube/pkg/helm/client"
	k8sclient "yunion.io/x/yunion-kube/pkg/k8s/client"
	k8sutil "yunion.io/x/yunion-kube/pkg/k8s/util"
	"yunion.io/x/yunion-kube/pkg/models/clusters"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/types"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

type Request struct {
	Cluster           *clusters.SCluster
	ClusterManager    *yclient.ClusterManager
	K8sClient         client.Interface
	K8sAdminClient    client.Interface
	K8sConfig         *rest.Config
	K8sAdminConfig    *rest.Config
	UserCred          mcclient.TokenCredential
	Query             *jsonutils.JSONDict
	Data              *jsonutils.JSONDict
	Context           context.Context
	KubeAdminConfig   string
	ProjectNamespaces *ProjectNamespaces
}

func (r *Request) AllowListItems() bool {
	allNamespace := jsonutils.QueryBoolean(r.Query, "all_namespace", false)
	if allNamespace && !r.UserCred.HasSystemAdminPrivilege() {
		return false
	}
	return true
}

func (r *Request) IsClusterOwner() bool {
	return r.UserCred.HasSystemAdminPrivilege() || r.GetCluster().IsOwner(r.UserCred)
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
		//namespace = r.GetProjectNamespace()
		namespace = v1.NamespaceDefault
	}
	return NewNamespaceQuery(namespace)
}

func (r *Request) GetK8sManager() *yclient.ClusterManager {
	return r.ClusterManager
}

func (r *Request) GetIndexer() *yclient.CacheFactory {
	return r.ClusterManager.GetIndexer()
}

func (r *Request) GetK8sClient() client.Interface {
	//return r.K8sClient
	return r.GetK8sAdminClient()
}

func (r *Request) GetK8sRestConfig() *rest.Config {
	//return r.K8sConfig
	return r.GetK8sAdminRestConfig()
}

func (r *Request) GetK8sAdminRestConfig() *rest.Config {
	return r.K8sAdminConfig
}

func (r *Request) GetK8sAdminClient() client.Interface {
	return r.K8sAdminClient
}

func (r *Request) GetCluster() *clusters.SCluster {
	return r.Cluster
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
	k8scli := r.GetK8sAdminClient()
	k8sconfig := r.GetK8sAdminRestConfig()
	return helmclient.NewHelmTunnelClient(k8scli, k8sconfig)
}

func (r *Request) GetGenericClient() (*k8sclient.GenericClient, error) {
	return k8sclient.NewGeneric(r.KubeAdminConfig)
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
	ns, err := r.Data.GetString("namespace")
	return ns, err
}

func (r *Request) GetDefaultNamespace() string {
	ns, _ := r.GetNamespaceByQuery()
	if ns != "" {
		return ns
	}
	ns, _ = r.GetNamespaceByData()
	if ns != "" {
		return ns
	}
	return v1.NamespaceDefault
}

func NewDataSelectQuery(query jsonutils.JSONObject) *dataselect.DataSelectQuery {
	limit, _ := query.Int("limit")
	if limit == 0 {
		limit = 20
	}
	offset, _ := query.Int("offset")
	limitQ := dataselect.NewLimitQuery(int(limit))
	offsetQ := dataselect.NewOffsetQuery(int(offset))

	filterQ := dataselect.NoFilter()
	filterRawCond := []string{}
	name, _ := query.GetString("name")
	if name != "" {
		filterRawCond = append(filterRawCond, dataselect.NameProperty, name)
	}
	namespace, _ := query.GetString("namespace")
	if namespace != "" {
		filterRawCond = append(filterRawCond, dataselect.NamespaceProperty, namespace)
	}
	if len(filterRawCond) != 0 {
		filterQ = dataselect.NewFilterQuery(filterRawCond)
	}
	sortQuery := dataselect.NewSortQuery([]string{"d", dataselect.CreationTimestampProperty})
	return dataselect.NewDataSelectQuery(
		sortQuery,
		filterQ,
		limitQ,
		offsetQ,
	)
}

//func (r *Request) IsSystemAdmin() bool {
//return r.UserCred.HasSystemAdminPrivilege()
//}

func (r *Request) ToQuery() *dataselect.DataSelectQuery {
	return NewDataSelectQuery(r.Query)
}

func (r *Request) GetParams() map[string]string {
	return appctx.AppContextParams(r.Context)
}

func (r *Request) IsK8sResourceExists(kind string, namespace string, id string) (bool, error) {
	cli, err := r.GetVerberClient()
	if err != nil {
		return false, err
	}
	isNamespace := true
	if namespace == "" {
		isNamespace = false
	}
	_, err = cli.Get(kind, isNamespace, namespace, id)
	if err == nil {
		return true, nil
	}
	if apierrors.IsNotFound(err) {
		return false, nil
	}
	return false, err
}

type ProjectNamespaces struct {
	Request    *Request
	namespaces []string
}

func newProjectNamespaces(req *Request) (*ProjectNamespaces, error) {
	nss, err := req.getProjectNamespaces()
	if err != nil {
		return nil, err
	}
	return &ProjectNamespaces{
		Request:    req,
		namespaces: nss,
	}, nil
}

func (pns *ProjectNamespaces) List() []string {
	return pns.namespaces
}

func (pns *ProjectNamespaces) Sets() sets.String {
	return sets.NewString(pns.namespaces...)
}

func (pns *ProjectNamespaces) HasAllNamespacePrivelege() bool {
	return pns.Request.UserCred.HasSystemAdminPrivilege()
}

func (r *Request) NewProjectNamespaces() (*ProjectNamespaces, error) {
	return newProjectNamespaces(r)
}

// TODO: support multiple namespace related to one project
func (r *Request) getProjectNamespaces() ([]string, error) {
	ns := types.ConvertProjectToNamespace(r.UserCred.GetProjectName())
	if len(ns) == 0 {
		return nil, httperrors.NewNotAcceptableError("Unsupport project name: %s", r.UserCred.GetProjectName())
	}
	return []string{ns}, nil
}

func (r *Request) EnsureProjectNamespaces() error {
	projectNamespaces, err := r.NewProjectNamespaces()
	if err != nil {
		return err
	}
	r.ProjectNamespaces = projectNamespaces
	return k8sutil.EnsureNamespaces(r.GetK8sAdminClient(), projectNamespaces.List()...)
}

func ValidateK8sResourceCreateData(req *Request, kind string, inNamespace bool) error {
	data := req.Data
	name, _ := data.GetString("name")
	if name == "" {
		return httperrors.NewInputParameterError("Name must provided")
	}
	namespace := ""
	if inNamespace {
		namespace, _ = req.GetNamespaceByData()
		if namespace == "" {
			namespace = req.GetDefaultNamespace()
			data.Set("namespace", jsonutils.NewString(namespace))
		}
	}

	exist, err := req.IsK8sResourceExists(kind, namespace, name)
	if err != nil {
		return err
	}
	if exist {
		return httperrors.NewDuplicateResourceError("Resource %s %s already exists", kind, name)
	}

	return nil
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
