package job

import (
	batch "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	"yunion.io/x/yunion-kube/pkg/resources/event"
	"yunion.io/x/yunion-kube/pkg/resources/pod"
)

// JobDetail is a presentation layer view of Kubernetes Job resource. This means
// it is Job plus additional augmented data we can get from other sources
// (like services that target the same pods).
type JobDetail struct {
	Job

	// Detailed information about Pods belonging to this Job.
	PodList []pod.Pod `json:"pods"`

	// List of events related to this Job.
	EventList []common.Event `json:"events"`
}

func (man *SJobManager) Get(req *common.Request, id string) (interface{}, error) {
	return GetJobDetail(req.GetK8sClient(), req.GetNamespaceQuery().ToRequestParam(), id)
}

func GetJobDetail(client kubernetes.Interface, namespace, name string) (*JobDetail, error) {
	jobData, err := client.BatchV1().Jobs(namespace).Get(name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	podList, err := GetJobPods(client, dataselect.DefaultDataSelect, namespace, name)
	if err != nil {
		return nil, err
	}

	podInfo, err := getJobPodInfo(client, jobData)
	if err != nil {
		return nil, err
	}

	eventList, err := GetJobEvents(client, dataselect.DefaultDataSelect, jobData.Namespace, jobData.Name)
	if err != nil {
		return nil, err
	}

	commonJob := toJob(jobData, podInfo)

	job := toJobDetail(commonJob, *eventList, *podList, *podInfo)
	return &job, nil
}

// GetJobPods return list of pods targeting job.
func GetJobPods(client kubernetes.Interface, dsQuery *dataselect.DataSelectQuery, namespace string, jobName string) (*pod.PodList, error) {
	log.Infof("Getting replication controller %s pods in namespace %s", jobName, namespace)

	pods, err := getRawJobPods(client, jobName, namespace)
	if err != nil {
		return nil, err
	}

	events, err := event.GetPodsEvents(client, namespace, pods)
	if err != nil {
		return nil, err
	}

	return pod.ToPodList(pods, events, dsQuery)
}

// Returns array of api pods targeting job with given name.
func getRawJobPods(client kubernetes.Interface, petSetName, namespace string) ([]v1.Pod, error) {
	job, err := client.Batch().Jobs(namespace).Get(petSetName, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	labelSelector := labels.SelectorFromSet(job.Spec.Selector.MatchLabels)
	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(client, common.NewSameNamespaceQuery(namespace),
			metaV1.ListOptions{
				LabelSelector: labelSelector.String(),
				FieldSelector: fields.Everything().String(),
			}),
	}

	podList := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	return podList.Items, nil
}

// Returns simple info about pods(running, desired, failing, etc.) related to given job.
func getJobPodInfo(client kubernetes.Interface, job *batch.Job) (*common.PodInfo, error) {
	labelSelector := labels.SelectorFromSet(job.Spec.Selector.MatchLabels)
	channels := &common.ResourceChannels{
		PodList: common.GetPodListChannelWithOptions(client, common.NewSameNamespaceQuery(
			job.Namespace),
			metaV1.ListOptions{
				LabelSelector: labelSelector.String(),
				FieldSelector: fields.Everything().String(),
			}),
	}

	pods := <-channels.PodList.List
	if err := <-channels.PodList.Error; err != nil {
		return nil, err
	}

	podInfo := common.GetPodInfo(job.Status.Active, job.Spec.Completions, pods.Items)

	// This pod info for jobs should be get from job status, similar to kubectl describe logic.
	podInfo.Running = job.Status.Active
	podInfo.Succeeded = job.Status.Succeeded
	podInfo.Failed = job.Status.Failed
	return &podInfo, nil
}

func toJobDetail(job Job, eventList common.EventList, podList pod.PodList, podInfo common.PodInfo) JobDetail {
	return JobDetail{
		Job:       job,
		PodList:   podList.Pods,
		EventList: eventList.Events,
	}
}

// GetJobEvents gets events associated to job.
func GetJobEvents(client kubernetes.Interface, dsQuery *dataselect.DataSelectQuery, namespace, name string) (*common.EventList, error) {
	jobEvents, err := event.GetEvents(client, namespace, name)
	if err != nil {
		return nil, err
	}

	list, err := event.CreateEventList(jobEvents, dsQuery)
	return &list, err
}
