package job

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// JobList contains a list of Jobs in the cluster.
type JobList struct {
	*common.BaseList
	// Basic information about resources status on the list.
	Status common.ResourceStatus

	// Unordered list of Jobs.
	Jobs []Job

	Pods   []v1.Pod
	Events []v1.Event
}

type JobStatusType string

const (
	// JobRunning means the job is still running.
	JobStatusRunning JobStatusType = "Running"
	// JobComplete means the job has completed its execution.
	JobStatusComplete JobStatusType = "Complete"
	// JobFailed means the job has failed its execution.
	JobStatusFailed JobStatusType = "Failed"
)

type JobStatus struct {
	// Short, machine understandable job status code.
	Status JobStatusType `json:"status"`
	// A human-readable description of the status of related job.
	Message string `json:"message"`
}

// Job is a presentation layer view of Kubernetes Job resource. This means it is Job plus additional
// augmented data we can get from other sources
type Job struct {
	api.ObjectMeta
	api.TypeMeta

	// Aggregate information about pods belonging to this Job.
	Pods common.PodInfo `json:"podsInfo"`

	// Container images of the Job.
	ContainerImages []string `json:"containerImages"`

	// Init Container images of the Job.
	InitContainerImages []string `json:"initContainerImages"`

	// number of parallel jobs defined.
	Parallelism *int32 `json:"parallelism"`

	// Completions specifies the desired number of successfully finished pods the job should be run with.
	Completions *int32 `json:"completions"`

	// JobStatus contains inferred job status based on job conditions
	JobStatus JobStatus `json:"jobStatus"`
	Status    string    `json:"status"`
}

func (man *SJobManager) List(req *common.Request) (common.ListResource, error) {
	return GetJobList(req.GetK8sClient(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

// GetJobList returns a list of all Jobs in the cluster.
func GetJobList(client kubernetes.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*JobList, error) {
	log.Infof("Getting list of all jobs in the cluster")

	channels := &common.ResourceChannels{
		JobList:   common.GetJobListChannel(client, nsQuery),
		PodList:   common.GetPodListChannel(client, nsQuery),
		EventList: common.GetEventListChannel(client, nsQuery),
	}

	return GetJobListFromChannels(channels, dsQuery, cluster)
}

// GetJobListFromChannels returns a list of all Jobs in the cluster reading required resource list once from the channels.
func GetJobListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*JobList, error) {
	jobs := <-channels.JobList.List
	err := <-channels.JobList.Error
	if err != nil {
		return nil, err
	}

	pods := <-channels.PodList.List
	err = <-channels.PodList.Error
	if err != nil {
		return nil, err
	}

	events := <-channels.EventList.List
	err = <-channels.EventList.Error
	if err != nil {
		return nil, err
	}

	jobList, err := ToJobList(jobs.Items, pods.Items, events.Items, dsQuery, cluster)
	if err != nil {
		return nil, err
	}
	jobList.Status = getStatus(jobs, pods.Items, events.Items)
	return jobList, nil
}

func ToJobList(jobs []batch.Job, pods []v1.Pod, events []v1.Event, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*JobList, error) {
	jobList := &JobList{
		BaseList: common.NewBaseList(cluster),
		Jobs:     make([]Job, 0),
		Pods:     pods,
		Events:   events,
	}

	err := dataselect.ToResourceList(
		jobList,
		jobs,
		dataselect.NewNamespaceDataCell,
		dsQuery)

	return jobList, err
}

func (l *JobList) Append(obj interface{}) {
	job := obj.(batch.Job)
	matchingPods := common.FilterPodsForJob(job, l.Pods)
	podInfo := common.GetPodInfo(job.Status.Active, job.Spec.Completions, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(l.Events, matchingPods)
	l.Jobs = append(l.Jobs, toJob(&job, &podInfo, l.GetCluster()))
}

func (l *JobList) GetResponseData() interface{} {
	return l.Jobs
}

func toJob(job *batch.Job, podInfo *common.PodInfo, cluster api.ICluster) Job {
	jobStatus := JobStatus{Status: JobStatusRunning}
	for _, condition := range job.Status.Conditions {
		if condition.Type == batch.JobComplete && condition.Status == v1.ConditionTrue {
			jobStatus.Status = JobStatusComplete
			break
		} else if condition.Type == batch.JobFailed && condition.Status == v1.ConditionTrue {
			jobStatus.Status = JobStatusFailed
			jobStatus.Message = condition.Message
			break
		}
	}
	return Job{
		ObjectMeta:          api.NewObjectMetaV2(job.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindJob),
		ContainerImages:     common.GetContainerImages(&job.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&job.Spec.Template.Spec),
		Pods:                *podInfo,
		JobStatus:           jobStatus,
		Parallelism:         job.Spec.Parallelism,
		Completions:         job.Spec.Completions,
		Status:              string(jobStatus.Status),
	}
}

func getStatus(list *batch.JobList, pods []v1.Pod, events []v1.Event) common.ResourceStatus {
	info := common.ResourceStatus{}
	if list == nil {
		return info
	}

	for _, job := range list.Items {
		matchingPods := common.FilterPodsForJob(job, pods)
		podInfo := common.GetPodInfo(job.Status.Active, job.Spec.Completions, matchingPods)
		warnings := event.GetPodsEventWarnings(events, matchingPods)

		if len(warnings) > 0 {
			info.Failed++
		} else if podInfo.Pending > 0 {
			info.Pending++
		} else if podInfo.Running > 0 {
			info.Running++
		} else {
			info.Succeeded++
		}
	}

	return info
}
