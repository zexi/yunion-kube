package release

import (
	"bytes"
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/proto/hapi/release"

	"yunion.io/x/log"

	k8sclient "yunion.io/x/yunion-kube/pkg/k8s/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/configmap"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/deployment"
	"yunion.io/x/yunion-kube/pkg/resources/ingress"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
	"yunion.io/x/yunion-kube/pkg/resources/secret"
	"yunion.io/x/yunion-kube/pkg/resources/service"
	"yunion.io/x/yunion-kube/pkg/resources/statefulset"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func GetReleaseResources(
	cli *k8sclient.GenericClient,
	cluster apis.ICluster,
	rls *release.Release) (map[string]interface{}, error) {
	namespace := rls.Namespace
	reader := bytes.NewBufferString(rls.Manifest)
	objs, err := cli.Get(namespace, reader)
	if err != nil {
		return nil, err
	}
	k8sCli, _ := cli.KubernetesClientSet()
	return convertRuntimeObjs(k8sCli, cluster, objs, namespace)
}

func convertRuntimeObjs(
	cli kubernetes.Interface,
	cluster apis.ICluster,
	objMap map[string][]runtime.Object,
	namespace string,
) (map[string]interface{}, error) {
	nsQuery := common.NewNamespaceQuery(namespace)
	ret := make(map[string]interface{})
	for kind, objs := range objMap {
		k, cObjs, err := processObjs(kind, cli, cluster, objs, nsQuery, dataselect.DefaultDataSelect())
		if err != nil {
			return nil, err
		}
		ret[k] = cObjs
	}
	return ret, nil
}

type IObjLister interface {
	ListV2(k8sCli kubernetes.Interface, cluster apis.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (common.ListResource, error)
}

func processObjs(
	kind string,
	cli kubernetes.Interface,
	cluster apis.ICluster,
	objs []runtime.Object,
	nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery,
) (string, interface{}, error) {
	var kindPlural string
	var ret interface{}
	var err error
	kindFuncMap := map[string]IObjLister{
		apis.ResourceKindPod:         pod.PodManager,
		apis.ResourceKindDeployment:  deployment.DeploymentManager,
		apis.ResourceKindStatefulSet: statefulset.StatefulSetManager,
		apis.ResourceKindService:     service.ServiceManager,
		apis.ResourceKindConfigMap:   configmap.ConfigMapManager,
		apis.ResourceKindIngress:     ingress.IngressManager,
		apis.ResourceKindSecret:      secret.SecretManager,
	}
	manager, ok := kindFuncMap[kind]
	if !ok {
		ret = objs
	} else {
		ret, err = processResources(cli, cluster, objs, nsQuery, dsQuery, manager)
	}
	kindPlural = transToKindPlural(kind)
	return kindPlural, ret, err
}

func transToKindPlural(kind string) string {
	ret, ok := apis.KindToAPIMapping[kind]
	if !ok {
		return kind
	}
	return ret.Resource
}

func processResources(
	cli kubernetes.Interface,
	cluster apis.ICluster,
	objs []runtime.Object,
	nsQuery *common.NamespaceQuery,
	dsQuery *dataselect.DataSelectQuery,
	ILister IObjLister,
) (interface{}, error) {
	list, err := ILister.ListV2(cli, cluster, nsQuery, dsQuery)
	if err != nil {
		log.Errorf("Get configmap list error: %v", err)
		return nil, err
	}
	ret := make([]interface{}, 0)
	dataV := reflect.ValueOf(list.GetResponseData())
	for i := 0; i < dataV.Len(); i++ {
		objV := dataV.Index(i)
		obj := objV.Interface()
		if runtimeObjsHas(objs, obj.(IObjectMeta)) {
			ret = append(ret, obj)
		}
	}
	return ret, nil
}

type IObjectMeta interface {
	GetName() string
}

func runtimeObjsHas(
	objs []runtime.Object,
	iObj IObjectMeta,
) bool {
	getName := func(obj runtime.Object) string {
		metaV := reflect.ValueOf(obj).Elem().FieldByName("ObjectMeta")
		meta := metaV.Interface().(metav1.ObjectMeta)
		return meta.Name
	}
	for _, obj := range objs {
		if getName(obj) == iObj.GetName() {
			return true
		}
	}
	return false
}
