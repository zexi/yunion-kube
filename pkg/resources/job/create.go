package job

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/resources/app"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/types/apis"
)

func (man *SJobManager) ValidateCreateData(req *common.Request) error {
	req.Data.Set("controllerType", jsonutils.NewString(apis.ResourceKindJob))
	return app.ValidateCreateData(req)
}

func (man *SJobManager) Create(req *common.Request) (interface{}, error) {
	return createJob(req)
}

func createJob(req *common.Request) (*batch.Job, error) {
	objMeta, _, err := common.GetK8sObjectCreateMetaWithLabel(req)
	if err != nil {
		return nil, err
	}
	input := &api.JobCreateInput{}
	if err := req.DataUnmarshal(input); err != nil {
		return nil, err
	}
	if len(input.Template.Spec.RestartPolicy) == 0 {
		input.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	input.Template.ObjectMeta = *objMeta
	job := &batch.Job{
		ObjectMeta: *objMeta,
		Spec:       input.JobSpec,
	}
	return req.GetK8sClient().BatchV1().Jobs(job.Namespace).Create(job)
}
