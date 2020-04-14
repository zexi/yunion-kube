package release

import (
	"bytes"

	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/resource"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/helm"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

func GetReleaseResources(
	cli *helm.Client, rel *release.Release,
	indexer *client.CacheFactory, cluster api.ICluster,
	clusterMan model.ICluster,
) (map[string][]interface{}, error) {
	cfg := cli.GetConfig()
	ress, err := cfg.KubeClient.Build(bytes.NewBufferString(rel.Manifest), true)
	if err != nil {
		return nil, err
	}
	ret := make(map[string][]interface{})
	ress.Visit(func(info *resource.Info, err error) error {
		gvk := info.Object.GetObjectKind().GroupVersionKind()
		man := model.GetK8SModelManagerByKind(gvk.Kind)
		if man == nil {
			log.Warningf("not fond %s manager", gvk.Kind)
			return nil
		}
		kindPlural := man.KeywordPlural()
		modelObj, err := model.NewK8SModelObject(man, clusterMan, info.Object)
		if err != nil {
			return err
		}
		obj, err := model.GetObject(modelObj)
		if err != nil {
			return err
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
