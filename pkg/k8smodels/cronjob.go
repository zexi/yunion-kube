package k8smodels

import (
	batch "k8s.io/api/batch/v1"
	batch2 "k8s.io/api/batch/v1beta1"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var (
	CronJobManager *SCronJobManager
)

func init() {
	CronJobManager = &SCronJobManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SCronJob{},
			"cronjob",
			"cronjobs"),
	}
	CronJobManager.SetVirtualObject(CronJobManager)
}

type SCronJobManager struct {
	model.SK8SNamespaceResourceBaseManager
}

type SCronJob struct {
	model.SK8SNamespaceResourceBase
}

func (m SCronJobManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameCronJob,
		Object:       &batch2.CronJob{},
		KindName:     api.KindNameCronJob,
	}
}

func (obj *SCronJob) GetRawCronJob() *batch2.CronJob {
	return obj.GetK8SObject().(*batch2.CronJob)
}

func (obj *SCronJob) GetAPIObject() (*api.CronJob, error) {
	cj := obj.GetRawCronJob()
	return &api.CronJob{
		ObjectMeta:   obj.GetObjectMeta(),
		TypeMeta:     obj.GetTypeMeta(),
		Schedule:     cj.Spec.Schedule,
		Suspend:      cj.Spec.Suspend,
		Active:       len(cj.Status.Active),
		LastSchedule: cj.Status.LastScheduleTime,
	}, nil
}

func (obj *SCronJob) GetEvents() ([]*api.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func filterJobsByOwnerUID(UID types.UID, jobs []*batch.Job) (matchingJobs []*batch.Job) {
	for _, j := range jobs {
		for _, i := range j.OwnerReferences {
			if i.UID == UID {
				matchingJobs = append(matchingJobs, j)
				break
			}
		}
	}
	return
}

func filterJobsByState(active bool, jobs []*batch.Job) (matchingJobs []*batch.Job) {
	for _, j := range jobs {
		if active && j.Status.Active > 0 {
			matchingJobs = append(matchingJobs, j)
		} else if !active && j.Status.Active == 0 {
			matchingJobs = append(matchingJobs, j)
		} else {
			//sup
		}
	}
	return
}

func (obj *SCronJob) GetRawJobs() ([]*batch.Job, error) {
	jobs, err := JobManager.GetRawJobs(obj.GetCluster(), obj.GetNamespace())
	if err != nil {
		return nil, err
	}
	return filterJobsByOwnerUID(obj.GetMetaObject().GetUID(), jobs), nil
}

func (obj *SCronJob) GetJobsByState(active bool) ([]*api.Job, error) {
	jobs, err := obj.GetRawJobs()
	if err != nil {
		return nil, err
	}
	jobs = filterJobsByState(active, jobs)
	return JobManager.GetAPIJobs(obj.GetCluster(), jobs)
}

func (obj *SCronJob) GetActiveJobs() ([]*api.Job, error) {
	return obj.GetJobsByState(true)
}

func (obj *SCronJob) GetInActiveJobs() ([]*api.Job, error) {

	return obj.GetJobsByState(false)
}

func (obj *SCronJob) GetAPIDetailObject() (*api.CronJobDetail, error) {
	cj := obj.GetRawCronJob()
	apiObj, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	activeJobs, err := obj.GetActiveJobs()
	if err != nil {
		return nil, err
	}
	inactiveJobs, err := obj.GetInActiveJobs()
	if err != nil {
		return nil, err
	}
	return &api.CronJobDetail{
		CronJob:                 *apiObj,
		ConcurrencyPolicy:       string(cj.Spec.ConcurrencyPolicy),
		StartingDeadLineSeconds: cj.Spec.StartingDeadlineSeconds,
		ActiveJobs:              activeJobs,
		InactiveJobs:            inactiveJobs,
		Events:                  events,
	}, nil
}

// TriggerCronJob manually triggers a cron job and creates a new job.
func (obj *SCronJob) TriggerCronJob() error {
	cronJob := obj.GetRawCronJob()

	annotations := make(map[string]string)
	annotations["cronjob.kubernetes.io/instantiate"] = "manual"

	labels := make(map[string]string)
	for k, v := range cronJob.Spec.JobTemplate.Labels {
		labels[k] = v
	}

	//job name cannot exceed DNS1053LabelMaxLength (52 characters)
	var newJobName string
	if len(cronJob.Name) < 42 {
		newJobName = cronJob.Name + "-manual-" + rand.String(3)
	} else {
		newJobName = cronJob.Name[0:41] + "-manual-" + rand.String(3)
	}

	jobToCreate := &batch.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:        newJobName,
			Namespace:   obj.GetNamespace(),
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: cronJob.Spec.JobTemplate.Spec,
	}

	cli := obj.GetCluster().GetHandler()
	_, err := cli.CreateV2(api.ResourceNameJob, obj.GetNamespace(), jobToCreate)
	if err != nil {
		return err
	}

	return nil
}

func (m *SCronJobManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input api.CronJobCreateInput) (runtime.Object, error) {
	if len(input.JobTemplate.Spec.Template.Spec.RestartPolicy) == 0 {
		input.JobTemplate.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	objMeta := input.ToObjectMeta()
	objMeta = *common.AddObjectMetaDefaultLabel(&objMeta)
	input.JobTemplate.Spec.Template.ObjectMeta = objMeta
	job := &batch2.CronJob{
		ObjectMeta: objMeta,
		Spec:       input.CronJobSpec,
	}
	return job, nil
}
