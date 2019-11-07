package release

import (
	"bytes"
	"strings"
	//"reflect"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/runtime"
	////"k8s.io/helm/pkg/proto/hapi/release"

	"yunion.io/x/log"

	//api "yunion.io/x/yunion-kube/pkg/apis"
	//k8sclient "yunion.io/x/yunion-kube/pkg/k8s/client"
	//"yunion.io/x/yunion-kube/pkg/resources/common"
	//"yunion.io/x/yunion-kube/pkg/resources/configmap"
	//"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	//"yunion.io/x/yunion-kube/pkg/resources/deployment"
	//"yunion.io/x/yunion-kube/pkg/resources/ingress"
	//"yunion.io/x/yunion-kube/pkg/resources/pod"
	//"yunion.io/x/yunion-kube/pkg/resources/secret"
	//"yunion.io/x/yunion-kube/pkg/resources/service"
	//"yunion.io/x/yunion-kube/pkg/resources/statefulset"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/resources"
)

func GetReleaseResources(
	cli *helm.Client, rel *release.Release,
	indexer *client.CacheFactory, cluster api.ICluster,
) (map[string][]interface{}, error) {
	cfg := cli.GetConfig()
	ress, err := cfg.KubeClient.Build(bytes.NewBufferString(rel.Manifest), true)
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]interface{})
	ress.Visit(func(info *resource.Info, err error) error {
		man := resources.KindManagerMap.Get(info.Object)
		var obj interface{}
		kindPlural := strings.ToLower(info.Object.GetObjectKind().GroupVersionKind().Kind)
		if man == nil {
			obj = info.Object
		} else {
			obj, err = convertRuntimeObj(indexer, cluster, info.Object, rel.Namespace)
			if err != nil {
				return err
			}
			kindPlural = man.KeywordPlural()
		}
		if list, ok := ret[kindPlural]; ok {
			list = append(list, obj)
		} else {
			list = []interface{}{obj}
			ret[kindPlural] = list
		}
		return nil
	})
	return ret, nil
}

type IObjectMeta interface {
	GetName() string
}

func convertRuntimeObj(
	cli *client.CacheFactory,
	cluster api.ICluster,
	obj runtime.Object,
	namespace string,
) (interface{}, error) {
	man := resources.KindManagerMap.Get(obj)
	log.Infof("=======Get manager %v, mans: %#v", man, resources.KindManagerMap)
	if man == nil {
		return obj, nil
	}
	return man.GetDetails(cli, cluster, namespace, obj.(IObjectMeta).GetName())
}
