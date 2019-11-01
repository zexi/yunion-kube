package cronjob

import (
	v1beta1 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SCronJobManager) ValidateCreateData(req *common.Request) error {
	req.Data.Set("controllerType", jsonutils.NewString(apis.ResourceKindCronJob))
	return app.ValidateCreateData(req)
}

func (man *SCronJobManager) Create(req *common.Request) (interface{}, error) {
	return createCronJob(req)
}

func createCronJob(req *common.Request) (*v1beta1.CronJob, error) {
	objMeta, _, err := common.GetK8sObjectCreateMetaWithLabel(req)
	if err != nil {
		return nil, err
	}
	input := &api.CronJobCreateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	if len(input.JobTemplate.Spec.Template.Spec.RestartPolicy) == 0 {
		input.JobTemplate.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	input.JobTemplate.Spec.Template.ObjectMeta = *objMeta

	job := &v1beta1.CronJob{
		ObjectMeta: *objMeta,
		Spec:       input.CronJobSpec,
	}
	return req.GetK8sClient().BatchV1beta1().CronJobs(job.GetNamespace()).Create(job)
}
