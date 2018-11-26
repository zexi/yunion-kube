package deployment

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

func (man *SDeploymentManager) ValidateCreateData(req *common.Request) error {
	req.Data.Set("controllerType", jsonutils.NewString(apis.ResourceKindDeployment))
	return app.ValidateCreateData(req)
}

func (man *SDeploymentManager) Create(req *common.Request) (interface{}, error) {
	return app.Create(req, createDeploymentApp)
}

func createDeploymentApp(
	cli client.Interface,
	objectMeta metaV1.ObjectMeta,
	labels map[string]string,
	podTemplate api.PodTemplateSpec,
	spec *app.AppDeploymentSpec,
) error {
	deployment := &apps.Deployment{
		ObjectMeta: objectMeta,
		Spec: apps.DeploymentSpec{
			Replicas: &spec.Replicas,
			Template: podTemplate,
			Selector: &metaV1.LabelSelector{
				// Quoting from https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#selector:
				// In API version apps/v1beta2, .spec.selector and .metadata.labels no longer default to
				// .spec.template.metadata.labels if not set. So they must be set explicitly.
				// Also note that .spec.selector is immutable after creation of the Deployment in apps/v1beta2.
				MatchLabels: labels,
			},
		},
	}
	_, err := cli.AppsV1beta2().Deployments(spec.Namespace).Create(deployment)
	return err
}
