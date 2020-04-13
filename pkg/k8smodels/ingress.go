package k8smodels

import (
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
)

var (
	IngressManager *SIngressManager
)

func init() {
	IngressManager = &SIngressManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			new(SIngress), "ingress", "ingresses"),
	}
}

type SIngressManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SIngress struct {
	model.SK8SNamespaceResourceBase
}

func (m *SIngressManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameIngress,
		Object:       new(extensions.Ingress),
	}
}

func (m *SIngressManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	_ *jsonutils.JSONDict,
	input *apis.IngressCreateInput) (runtime.Object, error) {
	objMeta := input.ToObjectMeta()
	ing := &extensions.Ingress{
		ObjectMeta: objMeta,
		Spec:       input.IngressSpec,
	}
	return ing, nil
}

func (obj *SIngress) GetRawIngress() *extensions.Ingress {
	return obj.GetK8SObject().(*extensions.Ingress)
}

func (obj *SIngress) getEndpoints(ingress *extensions.Ingress) []apis.Endpoint {
	endpoints := make([]apis.Endpoint, 0)
	if len(ingress.Status.LoadBalancer.Ingress) > 0 {
		for _, status := range ingress.Status.LoadBalancer.Ingress {
			endpoint := apis.Endpoint{Host: status.IP}
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func (obj *SIngress) GetAPIObject() (*apis.Ingress, error) {
	ing := obj.GetRawIngress()
	return &apis.Ingress{
		ObjectMeta: obj.GetObjectMeta(),
		TypeMeta:   obj.GetTypeMeta(),
		Endpoints:  obj.getEndpoints(ing),
	}, nil
}

func (obj *SIngress) GetAPIDetailObject() (*apis.IngressDetail, error) {
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	ing := obj.GetRawIngress()
	return &apis.IngressDetail{
		Ingress: *apiObj,
		Spec:    ing.Spec,
		Status:  ing.Status,
	}, nil
}
