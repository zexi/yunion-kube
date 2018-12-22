package statefulset

import (
	apps "k8s.io/api/apps/v1beta2"
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SStatefuleSetManager) ValidateCreateData(req *common.Request) error {
	req.Data.Set("controllerType", jsonutils.NewString(apis.ResourceKindStatefulSet))
	return app.ValidateCreateData(req)
}

func (man *SStatefuleSetManager) Create(req *common.Request) (interface{}, error) {
	return app.Create(req, createStatefulSetApp)
}

func createStatefulSetApp(
	cli client.Interface,
	objectMeta metaV1.ObjectMeta,
	labels map[string]string,
	podTemplate api.PodTemplateSpec,
	spec *app.AppDeploymentSpec,
) error {
	pvcs, err := spec.GetTemplatePVCs()
	if err != nil {
		return err
	}
	ss := &apps.StatefulSet{
		ObjectMeta: objectMeta,
		Spec: apps.StatefulSetSpec{
			Replicas:    &spec.Replicas,
			Template:    podTemplate,
			ServiceName: objectMeta.Name,
			Selector: &metaV1.LabelSelector{
				MatchLabels: labels,
			},
			VolumeClaimTemplates: pvcs,
		},
	}
	_, err = cli.AppsV1beta2().StatefulSets(spec.Namespace).Create(ss)
	return err
}
