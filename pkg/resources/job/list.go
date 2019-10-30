package job

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"

	"yunion.io/x/log"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
)

// JobList contains a list of Jobs in the cluster.
type JobList struct {
	*common.BaseList
	// Basic information about resources status on the list.
	Status common.ResourceStatus

	// Unordered list of Jobs.
	Jobs []api.Job

	Pods   []*v1.Pod
	Events []*v1.Event
}

func (man *SJobManager) List(req *common.Request) (common.ListResource, error) {
	return GetJobList(req.GetIndexer(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

// GetJobList returns a list of all Jobs in the cluster.
func GetJobList(indexer *client.CacheFactory, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*JobList, error) {
	log.Infof("Getting list of all jobs in the cluster")

	channels := &common.ResourceChannels{
		JobList:   common.GetJobListChannel(indexer, nsQuery),
		PodList:   common.GetPodListChannel(indexer, nsQuery),
		EventList: common.GetEventListChannel(indexer, nsQuery),
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

	jobList, err := ToJobList(jobs, pods, events, dsQuery, cluster)
	if err != nil {
		return nil, err
	}
	jobList.Status = getStatus(jobs, pods, events)
	return jobList, nil
}

func ToJobList(jobs []*batch.Job, pods []*v1.Pod, events []*v1.Event, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*JobList, error) {
	jobList := &JobList{
		BaseList: common.NewBaseList(cluster),
		Jobs:     make([]api.Job, 0),
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
	job := obj.(*batch.Job)
	matchingPods := common.FilterPodsForJob(job, l.Pods)
	podInfo := common.GetPodInfo(job.Status.Active, job.Spec.Completions, matchingPods)
	podInfo.Warnings = event.GetPodsEventWarnings(l.Events, matchingPods)
	l.Jobs = append(l.Jobs, toJob(job, &podInfo, l.GetCluster()))
}

func (l *JobList) GetResponseData() interface{} {
	return l.Jobs
}

func toJob(job *batch.Job, podInfo *api.PodInfo, cluster api.ICluster) api.Job {
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
	return api.Job{
		ObjectMeta:          api.NewObjectMeta(job.ObjectMeta, cluster),
		TypeMeta:            api.NewTypeMeta(job.TypeMeta),
		ContainerImages:     common.GetContainerImages(&job.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&job.Spec.Template.Spec),
		Pods:                *podInfo,
		JobStatus:           jobStatus,
		Parallelism:         job.Spec.Parallelism,
		Completions:         job.Spec.Completions,
		Status:              string(jobStatus.Status),
	}
}

func getStatus(list []*batch.Job, pods []*v1.Pod, events []*v1.Event) common.ResourceStatus {
	info := common.ResourceStatus{}
	if list == nil {
		return info
	}

	for _, job := range list {
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
