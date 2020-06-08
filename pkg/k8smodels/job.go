package k8smodels

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/api"
	"yunion.io/x/yunion-kube/pkg/k8s/common/model"
	"yunion.io/x/yunion-kube/pkg/resources/common"
)

var (
	JobManager *SJobManager
	_          model.IPodOwnerModel = new(SJob)
)

func init() {
	JobManager = &SJobManager{
		SK8SNamespaceResourceBaseManager: model.NewK8SNamespaceResourceBaseManager(
			&SJob{},
			"job",
			"jobs")}
	JobManager.SetVirtualObject(JobManager)
}

type SJobManager struct {
	model.SK8SNamespaceResourceBaseManager
	model.SK8SOwnerResourceBaseManager
}

type SJob struct {
	model.SK8SNamespaceResourceBase
}

func (m SJobManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: api.ResourceNameJob,
		Object:       &batch.Job{},
		KindName:     api.KindNameJob,
	}
}

func (m SJobManager) GetRawJobs(cluster model.ICluster, ns string) ([]*batch.Job, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.JobLister().Jobs(ns).List(labels.Everything())
}

func (m SJobManager) ListItemFilter(ctx *model.RequestContext, q model.IQuery, query *api.JobListInput) (model.IQuery, error) {
	q, err := m.SK8SNamespaceResourceBaseManager.ListItemFilter(ctx, q, query.ListInputK8SNamespaceBase)
	if err != nil {
		return q, err
	}
	q, err = m.SK8SOwnerResourceBaseManager.ListItemFilter(ctx, q, query.ListInputOwner)
	if err != nil {
		return q, err
	}
	if query.Active != nil {
		q.AddFilter(func(obj model.IK8SModel) (bool, error) {
			j := obj.(*SJob)
			rawJob := j.GetRawJob()
			isActive := *query.Active
			if isActive {
				return rawJob.Status.Active > 0, nil
			}
			return rawJob.Status.Active == 0, nil
		})
	}
	return q, nil
}

func (obj *SJob) IsOwnerBy(ownerModel model.IK8SModel) (bool, error) {
	return model.IsJobOwner(ownerModel, obj.GetRawJob())
}

func (obj *SJob) GetRawJob() *batch.Job {
	return obj.GetK8SObject().(*batch.Job)
}

func (obj *SJob) GetRawPods() ([]*v1.Pod, error) {
	job := obj.GetRawJob()
	indexer := obj.GetCluster().GetHandler().GetIndexer()
	labelSelector := labels.SelectorFromSet(job.Spec.Selector.MatchLabels)
	return indexer.PodLister().Pods(job.GetNamespace()).List(labelSelector)
}

func (obj *SJob) GetPods() ([]*api.Pod, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

func (obj *SJob) GetPodInfo(pods []*v1.Pod) (*api.PodInfo, error) {
	job := obj.GetRawJob()
	podInfo := common.GetPodInfo(job.Status.Active, job.Spec.Completions, pods)
	// This pod info for jobs should be get from job status, similar to kubectl describe logic.
	podInfo.Running = job.Status.Active
	podInfo.Succeeded = job.Status.Succeeded
	podInfo.Failed = job.Status.Failed
	return &podInfo, nil
}

func (obj *SJob) GetEvents() ([]*api.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SJob) GetAPIObject() (*api.Job, error) {
	job := obj.GetRawJob()
	jobStatus := api.JobStatus{Status: api.JobStatusRunning}
	for _, condition := range job.Status.Conditions {
		if condition.Type == batch.JobComplete && condition.Status == v1.ConditionTrue {
			jobStatus.Status = api.JobStatusComplete
			break
		} else if condition.Type == batch.JobFailed && condition.Status == v1.ConditionTrue {
			jobStatus.Status = api.JobStatusFailed
			jobStatus.Message = condition.Message
			break
		}
	}
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	podInfo, err := obj.GetPodInfo(pods)
	return &api.Job{
		ObjectMeta:          obj.GetObjectMeta(),
		TypeMeta:            obj.GetTypeMeta(),
		Pods:                podInfo,
		ContainerImages:     common.GetContainerImages(&job.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&job.Spec.Template.Spec),
		Parallelism:         job.Spec.Parallelism,
		Completions:         job.Spec.Completions,
		JobStatus:           jobStatus,
		Status:              string(jobStatus.Status),
	}, nil
}

func (obj *SJob) GetAPIDetailObject() (*api.JobDetail, error) {
	job, err := obj.GetAPIObject()
	if err != nil {
		return nil, err
	}
	pods, err := obj.GetPods()
	if err != nil {
		return nil, err
	}
	events, err := obj.GetEvents()
	if err != nil {
		return nil, err
	}
	return &api.JobDetail{
		Job:       *job,
		PodList:   pods,
		EventList: events,
	}, nil
}

func (m *SJobManager) ValidateCreateData(ctx *model.RequestContext, query *jsonutils.JSONDict, input *api.JobCreateInput) (*api.JobCreateInput, error) {
	if _, err := m.SK8SNamespaceResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sNamespaceResourceCreateInput); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SJobManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input api.JobCreateInput) (runtime.Object, error) {
	if len(input.Template.Spec.RestartPolicy) == 0 {
		input.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	job := &batch.Job{
		ObjectMeta: input.ToObjectMeta(),
		Spec:       input.JobSpec,
	}
	return job, nil
}

func (m *SJobManager) GetAPIJobs(cluster model.ICluster, jobs []*batch.Job) ([]*api.Job, error) {
	ret := make([]*api.Job, 0)
	err := ConvertRawToAPIObjects(m, cluster, jobs, &ret)
	return ret, err
}
