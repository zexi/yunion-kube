package common

import (
	apps "k8s.io/api/apps/v1beta2"
	autoscaling "k8s.io/api/autoscaling/v1"
	batch "k8s.io/api/batch/v1"
	batch2 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	storage "k8s.io/api/storage/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "k8s.io/client-go/kubernetes"

	api "yunion.io/x/yunion-kube/pkg/types/apis"
)

// ResourceChannels struct holds channels to resource lists. Each list channel is paired with
// an error channel which *must* be read sequentially: first read the list channel and then the error channel.
type ResourceChannels struct {
	// List and error channels to Replication Controllers.
	ReplicationControllerList ReplicationControllerListChannel

	// List and error channels to Replica Sets
	ReplicaSetList ReplicaSetListChannel

	// List and error channels to Deployments
	DeploymentList DeploymentListChannel

	// List and error channels to Daemon Sets
	DaemonSetList DaemonSetListChannel

	// List and error channels to Jobs.
	JobList JobListChannel

	// List and error channels to Cron Jobs.
	CronJobList CronJobListChannel

	// List and error channels to Services
	ServiceList ServiceListChannel

	// List and error channels to Endpoints.
	EndpointList EndpointListChannel

	// List and error channels to Ingresses.
	IngressList IngressListChannel

	// List and error channels to Pods
	PodList PodListChannel

	// List and error channels to Events.
	EventList EventListChannel

	// List and error channels to LimitRanges.
	LimitRangeList LimitRangeListChannel

	// List and error channels to Nodes.
	NodeList NodeListChannel

	// List and error channels to Namespaces.
	NamespaceList NamespaceListChannel

	// List and error channels to StatefulSets.
	StatefulSetList StatefulSetListChannel

	// List and error channels to ConfigMaps.
	ConfigMapList ConfigMapListChannel

	// List and error channels to Secrets.
	SecretList SecretListChannel

	// List and error channels to PersistentVolumes
	PersistentVolumeList PersistentVolumeListChannel

	// List and error channels to PersistentVolumeClaims
	PersistentVolumeClaimList PersistentVolumeClaimListChannel

	// List and error channels to ResourceQuotas
	ResourceQuotaList ResourceQuotaListChannel

	// List and error channels to HorizontalPodAutoscalers
	HorizontalPodAutoscalerList HorizontalPodAutoscalerListChannel

	// List and error channels to StorageClasses
	StorageClassList StorageClassListChannel

	// List and error channels to Roles
	RoleList RoleListChannel

	// List and error channels to ClusterRoles
	ClusterRoleList ClusterRoleListChannel

	// List and error channels to RoleBindings
	RoleBindingList RoleBindingListChannel

	// List and error channels to ClusterRoleBindings
	ClusterRoleBindingList ClusterRoleBindingListChannel
}

// IngressListChannel is a list and error channels to Ingresss.
type IngressListChannel struct {
	List  chan *extensions.IngressList
	Error chan error
}

func GetIngressListChannel(client client.Interface, nsQuery *NamespaceQuery) IngressListChannel {

	channel := IngressListChannel{
		List:  make(chan *extensions.IngressList),
		Error: make(chan error),
	}
	go func() {
		list, err := client.ExtensionsV1beta1().Ingresses(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []extensions.Ingress
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type ServiceListChannel struct {
	List  chan *v1.ServiceList
	Error chan error
}

func GetServiceListChannel(client client.Interface, nsQuery *NamespaceQuery) ServiceListChannel {
	channel := ServiceListChannel{
		List:  make(chan *v1.ServiceList),
		Error: make(chan error),
	}
	go func() {
		list, err := client.CoreV1().Services(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []v1.Service
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()
	return channel
}

type LimitRangeListChannel struct {
	List  chan *v1.LimitRangeList
	Error chan error
}

func GetLimitRangeListChannel(client client.Interface, nsQuery *NamespaceQuery) LimitRangeListChannel {

	channel := LimitRangeListChannel{
		List:  make(chan *v1.LimitRangeList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().LimitRanges(nsQuery.ToRequestParam()).List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type NodeListChannel struct {
	List  chan *v1.NodeList
	Error chan error
}

func GetNodeListChannel(client client.Interface) NodeListChannel {
	channel := NodeListChannel{
		List:  make(chan *v1.NodeList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Nodes().List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type NamespaceListChannel struct {
	List  chan *v1.NamespaceList
	Error chan error
}

func GetNamespaceListChannel(client client.Interface) NamespaceListChannel {
	channel := NamespaceListChannel{
		List:  make(chan *v1.NamespaceList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Namespaces().List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type EventListChannel struct {
	List  chan *v1.EventList
	Error chan error
}

func GetEventListChannel(client client.Interface, nsQuery *NamespaceQuery) EventListChannel {
	return GetEventListChannelWithOptions(client, nsQuery, api.ListEverything)
}

func GetEventListChannelWithOptions(client client.Interface,
	nsQuery *NamespaceQuery, options metaV1.ListOptions) EventListChannel {
	channel := EventListChannel{
		List:  make(chan *v1.EventList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Events(nsQuery.ToRequestParam()).List(options)
		var filteredItems []v1.Event
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type EndpointListChannel struct {
	List  chan *v1.EndpointsList
	Error chan error
}

func GetEndpointListChannel(client client.Interface, nsQuery *NamespaceQuery) EndpointListChannel {
	return GetEndpointListChannelWithOptions(client, nsQuery, api.ListEverything)
}

func GetEndpointListChannelWithOptions(client client.Interface,
	nsQuery *NamespaceQuery, opt metaV1.ListOptions) EndpointListChannel {
	channel := EndpointListChannel{
		List:  make(chan *v1.EndpointsList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Endpoints(nsQuery.ToRequestParam()).List(opt)

		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// PodListChannel is a list and error channels to Nodes
type PodListChannel struct {
	List  chan *v1.PodList
	Error chan error
}

func GetPodListChannel(client client.Interface, nsQuery *NamespaceQuery) PodListChannel {
	return GetPodListChannelWithOptions(client, nsQuery, api.ListEverything)
}

func GetPodListChannelWithOptions(client client.Interface, nsQuery *NamespaceQuery, options metaV1.ListOptions) PodListChannel {
	channel := PodListChannel{
		List:  make(chan *v1.PodList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Pods(nsQuery.ToRequestParam()).List(options)
		var filteredItems []v1.Pod
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type ReplicationControllerListChannel struct {
	List  chan *v1.ReplicationControllerList
	Error chan error
}

func GetReplicationControllerListChannel(client client.Interface,
	nsQuery *NamespaceQuery) ReplicationControllerListChannel {

	channel := ReplicationControllerListChannel{
		List:  make(chan *v1.ReplicationControllerList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().ReplicationControllers(nsQuery.ToRequestParam()).
			List(api.ListEverything)
		var filteredItems []v1.ReplicationController
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type DeploymentListChannel struct {
	List  chan *apps.DeploymentList
	Error chan error
}

func GetDeploymentListChannel(client client.Interface,
	nsQuery *NamespaceQuery) DeploymentListChannel {
	channel := DeploymentListChannel{
		List:  make(chan *apps.DeploymentList),
		Error: make(chan error),
	}
	go func() {
		list, err := client.AppsV1beta2().Deployments(nsQuery.ToRequestParam()).
			List(api.ListEverything)
		var filteredItems []apps.Deployment
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type ReplicaSetListChannel struct {
	List  chan *apps.ReplicaSetList
	Error chan error
}

func GetReplicaSetListChannel(client client.Interface,
	nsQuery *NamespaceQuery) ReplicaSetListChannel {
	return GetReplicaSetListChannelWithOptions(client, nsQuery, api.ListEverything)
}

func GetReplicaSetListChannelWithOptions(client client.Interface, nsQuery *NamespaceQuery,
	options metaV1.ListOptions) ReplicaSetListChannel {
	channel := ReplicaSetListChannel{
		List:  make(chan *apps.ReplicaSetList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.AppsV1beta2().ReplicaSets(nsQuery.ToRequestParam()).
			List(options)
		var filteredItems []apps.ReplicaSet
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type DaemonSetListChannel struct {
	List  chan *apps.DaemonSetList
	Error chan error
}

func GetDaemonSetListChannel(client client.Interface, nsQuery *NamespaceQuery) DaemonSetListChannel {
	channel := DaemonSetListChannel{
		List:  make(chan *apps.DaemonSetList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.AppsV1beta2().DaemonSets(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []apps.DaemonSet
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type JobListChannel struct {
	List  chan *batch.JobList
	Error chan error
}

func GetJobListChannel(client client.Interface,
	nsQuery *NamespaceQuery) JobListChannel {
	channel := JobListChannel{
		List:  make(chan *batch.JobList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.BatchV1().Jobs(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []batch.Job
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

type CronJobListChannel struct {
	List  chan *batch2.CronJobList
	Error chan error
}

func GetCronJobListChannel(client client.Interface, nsQuery *NamespaceQuery) CronJobListChannel {
	channel := CronJobListChannel{
		List:  make(chan *batch2.CronJobList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.BatchV1beta1().CronJobs(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []batch2.CronJob
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// StatefulSetListChannel is a list and error channels to Nodes.
type StatefulSetListChannel struct {
	List  chan *apps.StatefulSetList
	Error chan error
}

func GetStatefulSetListChannel(client client.Interface,
	nsQuery *NamespaceQuery) StatefulSetListChannel {
	channel := StatefulSetListChannel{
		List:  make(chan *apps.StatefulSetList),
		Error: make(chan error),
	}

	go func() {
		statefulSets, err := client.AppsV1beta2().StatefulSets(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []apps.StatefulSet
		for _, item := range statefulSets.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		statefulSets.Items = filteredItems
		channel.List <- statefulSets
		channel.Error <- err
	}()

	return channel
}

// ConfigMapListChannel is a list and error channels to ConfigMaps.
type ConfigMapListChannel struct {
	List  chan *v1.ConfigMapList
	Error chan error
}

func GetConfigMapListChannel(client client.Interface, nsQuery *NamespaceQuery) ConfigMapListChannel {

	channel := ConfigMapListChannel{
		List:  make(chan *v1.ConfigMapList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().ConfigMaps(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []v1.ConfigMap
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// SecretListChannel is a list and error channels to Secrets.
type SecretListChannel struct {
	List  chan *v1.SecretList
	Error chan error
}

func GetSecretListChannel(client client.Interface, nsQuery *NamespaceQuery) SecretListChannel {

	channel := SecretListChannel{
		List:  make(chan *v1.SecretList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().Secrets(nsQuery.ToRequestParam()).List(api.ListEverything)
		var filteredItems []v1.Secret
		for _, item := range list.Items {
			if nsQuery.Matches(item.ObjectMeta.Namespace) {
				filteredItems = append(filteredItems, item)
			}
		}
		list.Items = filteredItems
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// RoleListChannel is a list and error channels to Roles.
type RoleListChannel struct {
	List  chan *rbac.RoleList
	Error chan error
}

func GetRoleListChannel(client client.Interface) RoleListChannel {
	channel := RoleListChannel{
		List:  make(chan *rbac.RoleList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.RbacV1().Roles("").List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// ClusterRoleListChannel is a list and error channels to ClusterRoles.
type ClusterRoleListChannel struct {
	List  chan *rbac.ClusterRoleList
	Error chan error
}

func GetClusterRoleListChannel(client client.Interface) ClusterRoleListChannel {
	channel := ClusterRoleListChannel{
		List:  make(chan *rbac.ClusterRoleList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.RbacV1().ClusterRoles().List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// RoleBindingListChannel is a list and error channels to RoleBindings.
type RoleBindingListChannel struct {
	List  chan *rbac.RoleBindingList
	Error chan error
}

func GetRoleBindingListChannel(client client.Interface) RoleBindingListChannel {
	channel := RoleBindingListChannel{
		List:  make(chan *rbac.RoleBindingList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.RbacV1().RoleBindings("").List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// ClusterRoleBindingListChannel is a list and error channels to ClusterRoleBindings.
type ClusterRoleBindingListChannel struct {
	List  chan *rbac.ClusterRoleBindingList
	Error chan error
}

func GetClusterRoleBindingListChannel(client client.Interface) ClusterRoleBindingListChannel {
	channel := ClusterRoleBindingListChannel{
		List:  make(chan *rbac.ClusterRoleBindingList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.RbacV1().ClusterRoleBindings().List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// PersistentVolumeListChannel is a list and error channels to PersistentVolumes.
type PersistentVolumeListChannel struct {
	List  chan *v1.PersistentVolumeList
	Error chan error
}

func GetPersistentVolumeListChannel(client client.Interface) PersistentVolumeListChannel {
	channel := PersistentVolumeListChannel{
		List:  make(chan *v1.PersistentVolumeList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().PersistentVolumes().List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// PersistentVolumeClaimListChannel is a list and error channels to PersistentVolumeClaims.
type PersistentVolumeClaimListChannel struct {
	List  chan *v1.PersistentVolumeClaimList
	Error chan error
}

func GetPersistentVolumeClaimListChannel(client client.Interface, nsQuery *NamespaceQuery) PersistentVolumeClaimListChannel {

	channel := PersistentVolumeClaimListChannel{
		List:  make(chan *v1.PersistentVolumeClaimList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().PersistentVolumeClaims(nsQuery.ToRequestParam()).List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// ResourceQuotaListChannel is a list and error channels to ResourceQuotas.
type ResourceQuotaListChannel struct {
	List  chan *v1.ResourceQuotaList
	Error chan error
}

func GetResourceQuotaListChannel(client client.Interface, nsQuery *NamespaceQuery) ResourceQuotaListChannel {

	channel := ResourceQuotaListChannel{
		List:  make(chan *v1.ResourceQuotaList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.CoreV1().ResourceQuotas(nsQuery.ToRequestParam()).List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// HorizontalPodAutoscalerListChannel is a list and error channels.
type HorizontalPodAutoscalerListChannel struct {
	List  chan *autoscaling.HorizontalPodAutoscalerList
	Error chan error
}

func GetHorizontalPodAutoscalerListChannel(client client.Interface, nsQuery *NamespaceQuery) HorizontalPodAutoscalerListChannel {
	channel := HorizontalPodAutoscalerListChannel{
		List:  make(chan *autoscaling.HorizontalPodAutoscalerList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.AutoscalingV1().HorizontalPodAutoscalers(nsQuery.ToRequestParam()).
			List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}

// StorageClassListChannel is a list and error channels to storage classes.
type StorageClassListChannel struct {
	List  chan *storage.StorageClassList
	Error chan error
}

func GetStorageClassListChannel(client client.Interface) StorageClassListChannel {
	channel := StorageClassListChannel{
		List:  make(chan *storage.StorageClassList),
		Error: make(chan error),
	}

	go func() {
		list, err := client.StorageV1().StorageClasses().List(api.ListEverything)
		channel.List <- list
		channel.Error <- err
	}()

	return channel
}
