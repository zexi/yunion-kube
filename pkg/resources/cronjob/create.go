package cronjob

import (
	batchv1 "k8s.io/api/batch/v1"
	v1beta1 "k8s.io/api/batch/v1beta1"
	api "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SCronJobManager) ValidateCreateData(req *common.Request) error {
	req.Data.Set("controllerType", jsonutils.NewString(apis.ResourceKindCronJob))
	return app.ValidateCreateData(req)
}

func (man *SCronJobManager) Create(req *common.Request) (interface{}, error) {
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
		schedule, err := req.Data.GetString("schedule")
		if err != nil {
			return err
		}
		job := &v1beta1.CronJob{
			ObjectMeta: objectMeta,
			Spec: v1beta1.CronJobSpec{
				Schedule: schedule,
				JobTemplate: v1beta1.JobTemplateSpec{
					ObjectMeta: objectMeta,
					Spec: batchv1.JobSpec{
						Template:    podTemplate,
						Parallelism: &parallelism,
						//Selector: &metaV1.LabelSelector{
						//MatchLabels: labels,
						//},
					},
				},
			},
		}
		_, err = cli.BatchV1beta1().CronJobs(spec.Namespace).Create(job)
		return err
	}
}
