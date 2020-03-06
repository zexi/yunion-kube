package statefulset

import (
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

func (man *SStatefuleSetManager) ValidateCreateData(req *common.Request) error {
	return app.ValidateCreateData(req, man)
}

func (man *SStatefuleSetManager) Create(req *common.Request) (interface{}, error) {
	return createStatefulSetApp(req)
}

func createStatefulSetApp(req *common.Request) (*apps.StatefulSet, error) {
	objMeta, selector, err := common.GetK8sObjectCreateMetaWithLabel(req)
	if err != nil {
		return nil, err
	}
	input := &api.StatefulsetCreateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	input.Template.ObjectMeta = *objMeta
	input.Selector = selector
	input.ServiceName = objMeta.GetName()

	for i, p := range input.VolumeClaimTemplates {
		temp := p.DeepCopy()
		temp.SetNamespace(objMeta.GetNamespace())
		if len(temp.Spec.AccessModes) == 0 {
			temp.Spec.AccessModes = []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
		}
		input.VolumeClaimTemplates[i] = *temp
	}

	if _, err := common.CreateServiceIfNotExist(req, objMeta, input.Service); err != nil {
		return nil, err
	}

	ss := &apps.StatefulSet{
		ObjectMeta: *objMeta,
		Spec:       input.StatefulSetSpec,
	}
	return req.GetK8sClient().AppsV1().StatefulSets(ss.Namespace).Create(ss)
}
