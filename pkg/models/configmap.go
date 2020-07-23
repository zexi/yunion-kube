package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	ConfigMapManager *SConfigMapManager
	_                IPodOwnerModel = new(SConfigMap)
)

func init() {
	ConfigMapManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SConfigMapManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SConfigMap),
				"configmaps_tbl",
				"configmap",
				"configmaps",
				api.ResourceNameConfigMap,
				api.KindNameConfigMap,
				new(v1.ConfigMap),
			),
		}
	}).(*SConfigMapManager)
}

type SConfigMapManager struct {
	SNamespaceResourceBaseManager
}

type SConfigMap struct {
	SNamespaceResourceBase
}

func (m *SConfigMapManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.ConfigMapCreateInput) (*api.ConfigMapCreateInput, error) {
	if _, err := m.SNamespaceResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, &input.NamespaceResourceCreateInput); err != nil {
		return input, err
	}
	if len(input.Data) == 0 {
		return nil, httperrors.NewNotAcceptableError("data is empty")
	}
	return input, nil
}

func (m *SConfigMap) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, body jsonutils.JSONObject) (interface{}, error) {
	input := new(api.ConfigMapCreateInput)
	body.Unmarshal(input)
	return &v1.ConfigMap{
		ObjectMeta: input.ToObjectMeta(),
		Data:       input.Data,
	}, nil
}

func (obj *SConfigMap) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	cfgName := obj.GetName()
	rawPods, err := PodManager.GetRawPodsByObjectNamespace(cli, rawObj)
	if err != nil {
		return nil, err
	}
	mountPods := make([]*v1.Pod, 0)
	markMap := make(map[string]bool, 0)
	for _, pod := range rawPods {
		cfgs := GetPodConfigMapVolumes(pod)
		for _, cfg := range cfgs {
			if cfg.ConfigMap.Name == cfgName {
				if _, ok := markMap[pod.GetName()]; !ok {
					mountPods = append(mountPods, pod)
					markMap[pod.GetName()] = true
				}
			}
		}
	}
	return mountPods, err
}
