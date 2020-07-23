package models

import (
	"context"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/k8s/common/getters"
)

var (
	StatefulSetManager *SStatefulSetManager
	_                  IPodOwnerModel = new(SStatefulSet)
)

func init() {
	StatefulSetManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SStatefulSetManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SStatefulSet),
				"statefulsets_tbl",
				"statefulsets",
				"statefulset",
				api.ResourceNameStatefulSet,
				api.KindNameStatefulSet,
				new(apps.StatefulSet),
			),
		}
	}).(*SStatefulSetManager)
}

type SStatefulSetManager struct {
	SNamespaceResourceBaseManager
}

type SStatefulSet struct {
	SNamespaceResourceBase
}

func (m *SStatefulSetManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.StatefulsetCreateInputV2)
	data.Unmarshal(input)
	objMeta := input.ToObjectMeta()
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.Template.ObjectMeta = objMeta
	input.Selector = GetSelectorByObjectMeta(&objMeta)
	input.ServiceName = objMeta.GetName()

	for i, p := range input.VolumeClaimTemplates {
		temp := p.DeepCopy()
		temp.SetNamespace(objMeta.GetNamespace())
		if len(temp.Spec.AccessModes) == 0 {
			temp.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
		}
		input.VolumeClaimTemplates[i] = *temp
	}
	if _, err := CreateServiceIfNotExist(cli, &objMeta, input.Service); err != nil {
		return nil, err
	}
	ss := &apps.StatefulSet{
		ObjectMeta: objMeta,
		Spec:       input.StatefulSetSpec,
	}
	return ss, nil
}

func (obj *SStatefulSet) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (api.StatefulSetDetailV2, error) {
	return api.StatefulSetDetailV2{}, nil
}

func (obj *SStatefulSet) GetRawPods(cli *client.ClusterManager, rawObj runtime.Object) ([]*v1.Pod, error) {
	ss := rawObj.(*apps.StatefulSet)
	pods, err := PodManager.GetRawPods(cli, ss.GetNamespace())
	if err != nil {
		return nil, errors.Wrap(err, "statefulset get raw pods")
	}
	return FilterPodsByControllerRef(ss, pods), nil
}

func (obj *SStatefulSet) GetPodInfo(cli *client.ClusterManager, ss *apps.StatefulSet) (*api.PodInfo, error) {
	pods, err := obj.GetRawPods(cli, ss)
	if err != nil {
		return nil, err
	}
	return GetPodInfo(ss.Status.Replicas, ss.Spec.Replicas, pods)
}

func (obj *SStatefulSet) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	ss := k8sObj.(*apps.StatefulSet)
	detail := api.StatefulSetDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Replicas:                ss.Spec.Replicas,
		ContainerImages:         GetContainerImages(&ss.Spec.Template.Spec),
		InitContainerImages:     GetContainerImages(&ss.Spec.Template.Spec),
		Selector:                ss.Spec.Selector.MatchLabels,
	}
	podInfo, err := obj.GetPodInfo(cli, ss)
	if err != nil {
		log.Errorf("Get pod info by statefulset %s error: %v", obj.GetName(), err)
	} else {
		detail.Pods = *podInfo
		detail.StatefulSetStatus = *getters.GetStatefulSetStatus(podInfo, *ss)
	}
	return detail
}

// TODO: support update
