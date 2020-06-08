package models

import (
	"context"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/yunion-kube/pkg/api"
)

var (
	PodManager *SPodManager
)

func init() {
	PodManager = &SPodManager{
		SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
			&SPod{},
			"pods_tbl",
			"pod",
			"pods",
			api.ResourceNamePod,
			api.KindNamePod,
			new(v1.Pod),
		),
	}
	PodManager.SetVirtualObject(PodManager)
	RegisterK8sModelManager(PodManager)
}

type SPodManager struct {
	SNamespaceResourceBaseManager
}

type SPod struct {
	SNamespaceResourceBase
	NodeId string `width:"36" charset:"ascii" nullable:"false"`

	// CpuRequests is number of allocated milicores
	CpuRequests int64 `list:"user"`
	// CpuLimits is defined cpu limit
	CpuLimits int64 `list:"user"`

	// MemoryRequests
	MemoryRequests int64 `list:"user"`
	// MemoryLimits
	MemoryLimits int64
}

func (m *SPodManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return nil, httperrors.NewBadRequestError("Not support pod create")
}

func (m *SPodManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.NamespaceResourceListInput) (*sqlchemy.SQuery, error) {
	return m.SNamespaceResourceBaseManager.ListItemFilter(ctx, q, userCred, input)
}

func (p *SPod) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, isList bool) (*api.PodDetail, error) {
	return nil, nil
}

func (m *SPodManager) NewFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, cluster *SCluster, obj interface{}) (IClusterModel, error) {
	model, err := m.SNamespaceResourceBaseManager.NewFromRemoteObject(ctx, userCred, cluster, obj)
	if err != nil {
		return nil, err
	}
	kPod := obj.(*v1.Pod)
	podObj := model.(*SPod)
	nodeName := kPod.Spec.NodeName
	if nodeName != "" {
		nodeObj, err := NodeManager.GetByName(userCred, podObj.ClusterId, nodeName)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch pod's node by name: %s", nodeName)
		}
		podObj.NodeId = nodeObj.GetId()
	}
	return podObj, nil
}

// PodRequestsAndLimits returns a dictionary of all defined resources summed up for all
// containers of the pod.
func PodRequestsAndLimits(pod *v1.Pod) (reqs map[v1.ResourceName]resource.Quantity, limits map[v1.ResourceName]resource.Quantity, err error) {
	reqs, limits = map[v1.ResourceName]resource.Quantity{}, map[v1.ResourceName]resource.Quantity{}
	for _, container := range pod.Spec.Containers {
		for name, quantity := range container.Resources.Requests {
			if value, ok := reqs[name]; !ok {
				reqs[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				reqs[name] = value
			}
		}
		for name, quantity := range container.Resources.Limits {
			if value, ok := limits[name]; !ok {
				limits[name] = quantity.DeepCopy()
			} else {
				value.Add(quantity)
				limits[name] = value
			}
		}
	}
	// init containers define the minimum of any resource
	for _, container := range pod.Spec.InitContainers {
		for name, quantity := range container.Resources.Requests {
			value, ok := reqs[name]
			if !ok {
				reqs[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				reqs[name] = quantity.DeepCopy()
			}
		}
		for name, quantity := range container.Resources.Limits {
			value, ok := limits[name]
			if !ok {
				limits[name] = quantity.DeepCopy()
				continue
			}
			if quantity.Cmp(value) > 0 {
				limits[name] = quantity.DeepCopy()
			}
		}
	}
	return
}

func (p *SPod) UpdateFromRemoteObject(ctx context.Context, userCred mcclient.TokenCredential, extObj interface{}) error {
	if err := p.SNamespaceResourceBase.UpdateFromRemoteObject(ctx, userCred, extObj); err != nil {
		return err
	}
	k8sPod := extObj.(*v1.Pod)
	reqs, limits, err := PodRequestsAndLimits(k8sPod)
	if err != nil {
		return errors.Wrap(err, "get pod resource requests and limits")
	}
	cpuRequests, cpuLimits, memoryRequests, memoryLimits := reqs[v1.ResourceCPU],
		limits[v1.ResourceCPU], reqs[v1.ResourceMemory], limits[v1.ResourceMemory]
	p.CpuRequests = cpuRequests.MilliValue()
	p.CpuLimits = cpuLimits.MilliValue()
	p.MemoryRequests = memoryRequests.MilliValue()
	p.MemoryLimits = memoryLimits.MilliValue()
	return nil
}

func (p *SPod) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	p.SNamespaceResourceBase.PostDelete(p, ctx, userCred)
}

func (m *SPodManager) GetPodsByClusters(clusterIds []string) ([]SPod, error) {
	pods := make([]SPod, 0)
	if err := GetResourcesByClusters(m, clusterIds, &pods); err != nil {
		return nil, err
	}
	return pods, nil
}
