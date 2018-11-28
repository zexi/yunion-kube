package job

import (
	batch "k8s.io/api/batch/v1"
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SJobManager) ValidateCreateData(req *common.Request) error {
	req.Data.Set("controllerType", jsonutils.NewString(apis.ResourceKindJob))
	return app.ValidateCreateData(req)
}

func (man *SJobManager) Create(req *common.Request) (interface{}, error) {
	return app.Create(req, createJobAppFactory(req))
}

func createJobAppFactory(req *common.Request) app.CreateResourceFunc {
	return func(
		cli client.Interface,
		objectMeta metaV1.ObjectMeta,
		labels map[string]string,
		podTemplate api.PodTemplateSpec,
		spec *app.AppDeploymentSpec,
	) error {
		parallelismInt64, _ := req.Data.Int("parallelism")
		var parallelism int32 = 1
		if parallelismInt64 != 0 {
			parallelism = int32(parallelismInt64)
		}
		job := &batch.Job{
			ObjectMeta: objectMeta,
			Spec: batch.JobSpec{
				Template:    podTemplate,
				Parallelism: &parallelism,
				//Selector: &metaV1.LabelSelector{
				//MatchLabels: labels,
				//},
			},
		}
		_, err := cli.BatchV1().Jobs(spec.Namespace).Create(job)
		return err
	}
}
