package models

import (
	batch2 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/client"
)

var (
	CronJobManager *SCronJobManager
	_              IClusterModel = new(SCronJob)
)

func init() {
	CronJobManager = NewK8sNamespaceModelManager(func() ISyncableManager {
		return &SCronJobManager{
			SNamespaceResourceBaseManager: NewNamespaceResourceBaseManager(
				new(SCronJob),
				"cronjobs_tbl",
				"cronjob",
				"cronjobs",
				api.ResourceNameCronJob,
				api.KindNameCronJob,
				new(batch2.CronJob),
			),
		}
	}).(*SCronJobManager)
}

type SCronJobManager struct {
	SNamespaceResourceBaseManager
}

type SCronJob struct {
	SNamespaceResourceBase
}

func (m *SCronJobManager) NewRemoteObjectForCreate(model IClusterModel, cli *client.ClusterManager, data jsonutils.JSONObject) (interface{}, error) {
	input := new(api.CronJobCreateInputV2)
	data.Unmarshal(input)
	if len(input.JobTemplate.Spec.Template.Spec.RestartPolicy) == 0 {
		input.JobTemplate.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	objMeta := input.ToObjectMeta()
	objMeta = *AddObjectMetaDefaultLabel(&objMeta)
	input.JobTemplate.Spec.Template.ObjectMeta = objMeta
	job := &batch2.CronJob{
		ObjectMeta: objMeta,
		Spec:       input.CronJobSpec,
	}
	return job, nil
}

func (obj *SCronJob) GetDetails(cli *client.ClusterManager, base interface{}, k8sObj runtime.Object, isList bool) interface{} {
	cj := k8sObj.(*batch2.CronJob)
	detail := api.CronJobDetailV2{
		NamespaceResourceDetail: obj.SNamespaceResourceBase.GetDetails(cli, base, k8sObj, isList).(api.NamespaceResourceDetail),
		Schedule:                cj.Spec.Schedule,
		Suspend:                 cj.Spec.Suspend,
		Active:                  len(cj.Status.Active),
		LastSchedule:            cj.Status.LastScheduleTime,
		ConcurrencyPolicy:       string(cj.Spec.ConcurrencyPolicy),
		StartingDeadLineSeconds: cj.Spec.StartingDeadlineSeconds,
	}
	return detail
}
