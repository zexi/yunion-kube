package cronjob

import (
	batch "k8s.io/api/batch/v1"
	batch2 "k8s.io/api/batch/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"

	api "yunion.io/x/yunion-kube/pkg/apis"
	"yunion.io/x/yunion-kube/pkg/client"
	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/job"
)

func (man *SCronJobManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetCronJobDetail(req.GetIndexer(), req.GetCluster(), req.ToQuery(), req.GetNamespaceQuery().ToRequestParam(), id)
}

// GetCronJobDetail gets Cron Job details.
func GetCronJobDetail(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*api.CronJobDetail, error) {
	rawObject, err := indexer.CronJobLister().CronJobs(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	activeJobs, err := GetCronJobJobs(indexer, cluster, dsQuery, namespace, name)
	if err != nil {
		return nil, err
	}

	inactiveJobs, err := GetCronJobCompletedJobs(indexer, cluster, dsQuery, namespace, name)

	events, err := GetCronJobEvents(indexer, cluster, dsQuery, namespace, name)
	if err != nil {
		return nil, err
	}

	cj := toCronJobDetail(rawObject, *activeJobs, *inactiveJobs, *events, cluster)
	return &cj, nil
}

func toCronJobDetail(cj *batch2.CronJob, activeJobs job.JobList, inactiveJobs job.JobList, events common.EventList, cluster api.ICluster) api.CronJobDetail {
	return api.CronJobDetail{
		CronJob:                 toCronJob(cj, cluster),
		ConcurrencyPolicy:       string(cj.Spec.ConcurrencyPolicy),
		StartingDeadLineSeconds: cj.Spec.StartingDeadlineSeconds,
		ActiveJobs:              activeJobs.Jobs,
		InactiveJobs:            inactiveJobs.Jobs,
		Events:                  events.Events,
	}
}

// GetCronJobJobs returns list of jobs owned by cron job.
func GetCronJobJobs(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*job.JobList, error) {

	cronJob, err := indexer.CronJobLister().CronJobs(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		JobList:   common.GetJobListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		PodList:   common.GetPodListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		EventList: common.GetEventListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}

	jobs := <-channels.JobList.List
	err = <-channels.JobList.Error
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

	jobs = filterJobsByOwnerUID(cronJob.UID, jobs)
	jobs = filterJobsByState(true, jobs)

	return job.ToJobList(jobs, pods, events, dsQuery, cluster)
}

// GetCronJobJobs returns list of jobs owned by cron job.
func GetCronJobCompletedJobs(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*job.JobList, error) {
	var err error

	cronJob, err := indexer.CronJobLister().CronJobs(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	channels := &common.ResourceChannels{
		JobList:   common.GetJobListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		PodList:   common.GetPodListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
		EventList: common.GetEventListChannel(indexer, common.NewSameNamespaceQuery(namespace)),
	}

	jobs := <-channels.JobList.List
	err = <-channels.JobList.Error
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

	jobs = filterJobsByOwnerUID(cronJob.UID, jobs)
	jobs = filterJobsByState(false, jobs)

	return job.ToJobList(jobs, pods, events, dsQuery, cluster)
}

// TriggerCronJob manually triggers a cron job and creates a new job.
func TriggerCronJob(client kubernetes.Interface, namespace, name string) error {
	cronJob, err := client.BatchV1beta1().CronJobs(namespace).Get(name, metaV1.GetOptions{})

	if err != nil {
		return err
	}

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
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: cronJob.Spec.JobTemplate.Spec,
	}

	_, err = client.BatchV1().Jobs(namespace).Create(jobToCreate)

	if err != nil {
		return err
	}

	return nil
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

// GetCronJobEvents gets events associated to cron job.
func GetCronJobEvents(indexer *client.CacheFactory, cluster api.ICluster, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*common.EventList, error) {
	raw, err := event.GetEvents(indexer, namespace, name)
	if err != nil {
		return nil, err
	}

	events, err := event.CreateEventList(raw, dsQuery, cluster)
	return events, err
}
