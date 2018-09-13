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
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// JobDetail is a presentation layer view of Kubernetes Job resource. This means
// it is Job plus additional augmented data we can get from other sources
// (like services that target the same pods).
type JobDetail struct {
	api.ObjectMeta
	api.TypeMeta

	// Aggregate information about pods belonging to this Job.
	PodInfo common.PodInfo `json:"podInfo"`

	// Detailed information about Pods belonging to this Job.
	PodList pod.PodList `json:"podList"`

	// Container images of the Job.
	ContainerImages []string `json:"containerImages"`

	// Init container images of the Job.
	InitContainerImages []string `json:"initContainerImages"`

	// List of events related to this Job.
	EventList common.EventList `json:"eventList"`

	// Parallelism specifies the maximum desired number of pods the job should run at any given time.
	Parallelism *int32 `json:"parallelism"`

	// Completions specifies the desired number of successfully finished pods the job should be run with.
	Completions *int32 `json:"completions"`
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

	job := toJobDetail(jobData, *eventList, *podList, *podInfo)
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

func toJobDetail(job *batch.Job, eventList common.EventList, podList pod.PodList, podInfo common.PodInfo) JobDetail {
	return JobDetail{
		ObjectMeta:          api.NewObjectMeta(job.ObjectMeta),
		TypeMeta:            api.NewTypeMeta(api.ResourceKindJob),
		ContainerImages:     common.GetContainerImages(&job.Spec.Template.Spec),
		InitContainerImages: common.GetInitContainerImages(&job.Spec.Template.Spec),
		PodInfo:             podInfo,
		PodList:             podList,
		EventList:           eventList,
		Parallelism:         job.Spec.Parallelism,
		Completions:         job.Spec.Completions,
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
