package k8smodels

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"yunion.io/x/jsonutils"

	"yunion.io/x/yunion-kube/pkg/apis"
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
}

type SJob struct {
	model.SK8SNamespaceResourceBase
}

func (m SJobManager) GetK8SResourceInfo() model.K8SResourceInfo {
	return model.K8SResourceInfo{
		ResourceName: apis.ResourceNameJob,
		Object:       &batch.Job{},
		KindName:     apis.KindNameJob,
	}
}

func (m SJobManager) GetRawJobs(cluster model.ICluster, ns string) ([]*batch.Job, error) {
	indexer := cluster.GetHandler().GetIndexer()
	return indexer.JobLister().Jobs(ns).List(labels.Everything())
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

func (obj *SJob) GetPods() ([]*apis.Pod, error) {
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	return PodManager.GetAPIPods(obj.GetCluster(), pods)
}

func (obj *SJob) GetPodInfo(pods []*v1.Pod) (*apis.PodInfo, error) {
	job := obj.GetRawJob()
	podInfo := common.GetPodInfo(job.Status.Active, job.Spec.Completions, pods)
	// This pod info for jobs should be get from job status, similar to kubectl describe logic.
	podInfo.Running = job.Status.Active
	podInfo.Succeeded = job.Status.Succeeded
	podInfo.Failed = job.Status.Failed
	return &podInfo, nil
}

func (obj *SJob) GetEvents() ([]*apis.Event, error) {
	return EventManager.GetEventsByObject(obj)
}

func (obj *SJob) GetAPIObject() (*apis.Job, error) {
	job := obj.GetRawJob()
	jobStatus := apis.JobStatus{Status: apis.JobStatusRunning}
	for _, condition := range job.Status.Conditions {
		if condition.Type == batch.JobComplete && condition.Status == v1.ConditionTrue {
			jobStatus.Status = apis.JobStatusComplete
			break
		} else if condition.Type == batch.JobFailed && condition.Status == v1.ConditionTrue {
			jobStatus.Status = apis.JobStatusFailed
			jobStatus.Message = condition.Message
			break
		}
	}
	pods, err := obj.GetRawPods()
	if err != nil {
		return nil, err
	}
	podInfo, err := obj.GetPodInfo(pods)
	return &apis.Job{
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

func (obj *SJob) GetAPIDetailObject() (*apis.JobDetail, error) {
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
	return &apis.JobDetail{
		Job:       *job,
		PodList:   pods,
		EventList: events,
	}, nil
}

func (m *SJobManager) ValidateCreateData(ctx *model.RequestContext, query *jsonutils.JSONDict, input *apis.JobCreateInput) (*apis.JobCreateInput, error) {
	if _, err := m.SK8SNamespaceResourceBaseManager.ValidateCreateData(ctx, query, &input.K8sNamespaceResourceCreateInput); err != nil {
		return nil, err
	}
	return input, nil
}

func (m *SJobManager) NewK8SRawObjectForCreate(
	ctx *model.RequestContext,
	input apis.JobCreateInput) (runtime.Object, error) {
	if len(input.Template.Spec.RestartPolicy) == 0 {
		input.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
	}
	job := &batch.Job{
		ObjectMeta: input.ToObjectMeta(),
		Spec:       input.JobSpec,
	}
	return job, nil
}

func (m *SJobManager) GetAPIJobs(cluster model.ICluster, jobs []*batch.Job) ([]*apis.Job, error) {
	ret := make([]*apis.Job, 0)
	err := ConvertRawToAPIObjects(m, cluster, jobs, &ret)
	return ret, err
}
