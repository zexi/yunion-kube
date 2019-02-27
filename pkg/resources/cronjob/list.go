package cronjob

import (
	"k8s.io/api/batch/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"yunion.io/x/log"

	"yunion.io/x/yunion-kube/pkg/resources/common"
	"yunion.io/x/yunion-kube/pkg/resources/dataselect"
	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// CronJobList contains a list of CronJobs in the cluster.
type CronJobList struct {
	*common.BaseList
	Items []CronJob

	// Basic information about resources status on the list.
	Status common.ResourceStatus `json:"status"`
}

// CronJob is a presentation layer view of Kubernetes Cron Job resource.
type CronJob struct {
	api.ObjectMeta
	api.TypeMeta
	Schedule     string       `json:"schedule"`
	Suspend      *bool        `json:"suspend"`
	Active       int          `json:"active"`
	LastSchedule *metav1.Time `json:"lastSchedule"`
}

func (man *SCronJobManager) List(req *common.Request) (common.ListResource, error) {
	return GetCronJobList(req.GetK8sClient(), req.GetCluster(), req.GetNamespaceQuery(), req.ToQuery())
}

// GetCronJobList returns a list of all CronJobs in the cluster.
func GetCronJobList(client kubernetes.Interface, cluster api.ICluster, nsQuery *common.NamespaceQuery, dsQuery *dataselect.DataSelectQuery) (*CronJobList, error) {
	log.Infof("Getting list of all cron jobs in the cluster")

	channels := &common.ResourceChannels{
		CronJobList: common.GetCronJobListChannel(client, nsQuery),
	}

	return GetCronJobListFromChannels(channels, dsQuery, cluster)
}

// GetCronJobListFromChannels returns a list of all CronJobs in the cluster reading required resource
// list once from the channels.
func GetCronJobListFromChannels(channels *common.ResourceChannels, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*CronJobList, error) {

	cronJobs := <-channels.CronJobList.List
	err := <-channels.CronJobList.Error
	if err != nil {
		return nil, err
	}

	cronJobList, err := toCronJobList(cronJobs.Items, dsQuery, cluster)
	if err != nil {
		return nil, err
	}
	cronJobList.Status = getStatus(cronJobs)
	return cronJobList, nil
}

func toCronJobList(cronJobs []v1beta1.CronJob, dsQuery *dataselect.DataSelectQuery, cluster api.ICluster) (*CronJobList, error) {
	list := &CronJobList{
		BaseList: common.NewBaseList(cluster),
		Items:    make([]CronJob, 0),
	}

	err := dataselect.ToResourceList(
		list,
		cronJobs,
		dataselect.NewNamespaceDataCell,
		dsQuery)

	return list, err
}

func (l *CronJobList) Append(obj interface{}) {
	cronJob := obj.(v1beta1.CronJob)
	l.Items = append(l.Items, toCronJob(&cronJob, l.GetCluster()))
}

func (l *CronJobList) GetResponseData() interface{} {
	return l.Items
}

func getStatus(list *v1beta1.CronJobList) common.ResourceStatus {
	info := common.ResourceStatus{}
	if list == nil {
		return info
	}

	for _, cronJob := range list.Items {
		if cronJob.Spec.Suspend != nil && !(*cronJob.Spec.Suspend) {
			info.Running++
		} else {
			info.Failed++
		}
	}

	return info
}

func toCronJob(cj *v1beta1.CronJob, cluster api.ICluster) CronJob {
	return CronJob{
		ObjectMeta:   api.NewObjectMetaV2(cj.ObjectMeta, cluster),
		TypeMeta:     api.NewTypeMeta(api.ResourceKindCronJob),
		Schedule:     cj.Spec.Schedule,
		Suspend:      cj.Spec.Suspend,
		Active:       len(cj.Status.Active),
		LastSchedule: cj.Status.LastScheduleTime,
	}
}
