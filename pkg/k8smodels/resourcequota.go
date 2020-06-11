package k8smodels

import (
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	ResourceQuotaManager *SResourceQuotaManager
	_                    model.IK8SModel = &SResourceQuota{}
)

func init() {
	ResourceQuotaManager = &SResourceQuotaManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SResourceQuota{},
			"resourcequota",
			"resourcequotas"),
	}
	ResourceQuotaManager.SetVirtualObject(ResourceQuotaManager)
}

type SResourceQuotaManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SResourceQuota struct {
	model.SK8SNamespaceResourceBase
}

func (m *SResourceQuotaManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameResourceQuota,
		Object:       &v1.ResourceQuota{},
		KindName:     api.KindNameResourceQuota,
	}
}

func (m *SResourceQuotaManager) GetRawResourceQuotas(cluster model.ICluster, ns string) ([]*v1.ResourceQuota, error) {
	return cluster.GetHandler().GetIndexer().
		ResourceQuotaLister().ResourceQuotas(ns).List(labels.Everything())
}

func (m *SResourceQuotaManager) GetResourceQuotaDetails(cluster model.ICluster, ns string) ([]*api.ResourceQuotaDetail, error) {
	rss, err := m.GetRawResourceQuotas(cluster, ns)
	if err != nil {
		return nil, err
	}
	ret := make([]*api.ResourceQuotaDetail, len(rss))
	for idx := range rss {
		rs, err := m.GetResourceQuotaDetail(cluster, rss[idx])
		if err != nil {
			return nil, err
		}
		ret[idx] = rs
	}
	return ret, nil
}

func (m *SResourceQuotaManager) GetResourceQuotaDetail(cluster model.ICluster, rs *v1.ResourceQuota) (*api.ResourceQuotaDetail, error) {
	mObj, err := model.NewK8SModelObject(m, cluster, rs)
	if err != nil {
		return nil, err
	}
	return mObj.(*SResourceQuota).GetAPIDetailObject()
}

func (obj *SResourceQuota) GetRawResourceQuota() *v1.ResourceQuota {
	return obj.GetK8SObject().(*v1.ResourceQuota)
}

func (obj *SResourceQuota) GetAPIObject() (*api.ResourceQuota, error) {
	rq := obj.GetRawResourceQuota()
	return &api.ResourceQuota{
		ObjectMeta:        obj.GetObjectMeta(),
		TypeMeta:          obj.GetTypeMeta(),
		ResourceQuotaSpec: rq.Spec,
	}, nil
}

func (obj *SResourceQuota) GetAPIDetailObject() (*api.ResourceQuotaDetail, error) {
	rs := obj.GetRawResourceQuota()
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}

	statusList := make(map[v1.ResourceName]api.ResourceStatus)
	for key, value := range rs.Status.Hard {
		used := rs.Status.Used[key]
		statusList[key] = api.ResourceStatus{
			Used: used.String(),
			Hard: value.String(),
		}
	}
	return &api.ResourceQuotaDetail{
		ResourceQuota: *apiObj,
		StatusList:    statusList,
	}, nil
}
