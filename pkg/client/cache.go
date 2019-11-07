package client

import (
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	apps "k8s.io/client-go/listers/apps/v1beta2"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	batch "k8s.io/client-go/listers/batch/v1"
	batch2 "k8s.io/client-go/listers/batch/v1beta1"
	"k8s.io/client-go/listers/core/v1"
	extensions "k8s.io/client-go/listers/extensions/v1beta1"
	rbac "k8s.io/client-go/listers/rbac/v1"
	storage "k8s.io/client-go/listers/storage/v1"

	"yunion.io/x/yunion-kube/pkg/client/api"
)

type CacheFactory struct {
	stopChan              chan struct{}
	sharedInformerFactory informers.SharedInformerFactory
}

func buildCacheController(client *kubernetes.Clientset) (*CacheFactory, error) {
	stop := make(chan struct{})
	sharedInformerFactory := informers.NewSharedInformerFactory(client, defaultResyncPeriod)

	// Start all Resources defined in KindToResourceMap
	for _, value := range api.KindToResourceMap {
		genericInformer, err := sharedInformerFactory.ForResource(value.GroupVersionResourceKind.GroupVersionResource)
		if err != nil {
			return nil, err
		}
		go genericInformer.Informer().Run(stop)
	}

	sharedInformerFactory.Start(stop)

	return &CacheFactory{
		stopChan:              stop,
		sharedInformerFactory: sharedInformerFactory,
	}, nil
}

func (c *CacheFactory) PodLister() v1.PodLister {
	return c.sharedInformerFactory.Core().V1().Pods().Lister()
}

func (c *CacheFactory) EventLister() v1.EventLister {
	return c.sharedInformerFactory.Core().V1().Events().Lister()
}

func (c *CacheFactory) ConfigMapLister() v1.ConfigMapLister {
	return c.sharedInformerFactory.Core().V1().ConfigMaps().Lister()
}

func (c *CacheFactory) SecretLister() v1.SecretLister {
	return c.sharedInformerFactory.Core().V1().Secrets().Lister()
}

func (c *CacheFactory) DeploymentLister() apps.DeploymentLister {
	return c.sharedInformerFactory.Apps().V1beta2().Deployments().Lister()
}

func (c *CacheFactory) DaemonSetLister() apps.DaemonSetLister {
	return c.sharedInformerFactory.Apps().V1beta2().DaemonSets().Lister()
}

func (c *CacheFactory) StatefulSetLister() apps.StatefulSetLister {
	return c.sharedInformerFactory.Apps().V1beta2().StatefulSets().Lister()
}

func (c *CacheFactory) NodeLister() v1.NodeLister {
	return c.sharedInformerFactory.Core().V1().Nodes().Lister()
}

func (c *CacheFactory) EndpointLister() v1.EndpointsLister {
	return c.sharedInformerFactory.Core().V1().Endpoints().Lister()
}

func (c *CacheFactory) HPALister() autoscalingv1.HorizontalPodAutoscalerLister {
	return c.sharedInformerFactory.Autoscaling().V1().HorizontalPodAutoscalers().Lister()
}

func (c *CacheFactory) IngressLister() extensions.IngressLister {
	return c.sharedInformerFactory.Extensions().V1beta1().Ingresses().Lister()
}

func (c *CacheFactory) ServiceLister() v1.ServiceLister {
	return c.sharedInformerFactory.Core().V1().Services().Lister()
}

func (c *CacheFactory) LimitRangeLister() v1.LimitRangeLister {
	return c.sharedInformerFactory.Core().V1().LimitRanges().Lister()
}

func (c *CacheFactory) NamespaceLister() v1.NamespaceLister {
	return c.sharedInformerFactory.Core().V1().Namespaces().Lister()
}

func (c *CacheFactory) ReplicationControllerLister() v1.ReplicationControllerLister {
	return c.sharedInformerFactory.Core().V1().ReplicationControllers().Lister()
}

func (c *CacheFactory) ReplicaSetLister() apps.ReplicaSetLister {
	return c.sharedInformerFactory.Apps().V1beta2().ReplicaSets().Lister()
}

func (c *CacheFactory) JobLister() batch.JobLister {
	return c.sharedInformerFactory.Batch().V1().Jobs().Lister()
}

func (c *CacheFactory) CronJobLister() batch2.CronJobLister {
	return c.sharedInformerFactory.Batch().V1beta1().CronJobs().Lister()
}

func (c *CacheFactory) PVLister() v1.PersistentVolumeLister {
	return c.sharedInformerFactory.Core().V1().PersistentVolumes().Lister()
}

func (c *CacheFactory) PVCLister() v1.PersistentVolumeClaimLister {
	return c.sharedInformerFactory.Core().V1().PersistentVolumeClaims().Lister()
}

func (c *CacheFactory) StorageClassLister() storage.StorageClassLister {
	return c.sharedInformerFactory.Storage().V1().StorageClasses().Lister()
}

func (c *CacheFactory) ResourceQuotaLister() v1.ResourceQuotaLister {
	return c.sharedInformerFactory.Core().V1().ResourceQuotas().Lister()
}

func (c *CacheFactory) RoleLister() rbac.RoleLister {
	return c.sharedInformerFactory.Rbac().V1().Roles().Lister()
}

func (c *CacheFactory) ClusterRoleLister() rbac.ClusterRoleLister {
	return c.sharedInformerFactory.Rbac().V1().ClusterRoles().Lister()
}

func (c *CacheFactory) RoleBindingLister() rbac.RoleBindingLister {
	return c.sharedInformerFactory.Rbac().V1().RoleBindings().Lister()
}

func (c *CacheFactory) ClusterRoleBindingLister() rbac.ClusterRoleBindingLister {
	return c.sharedInformerFactory.Rbac().V1().ClusterRoleBindings().Lister()
}

func (c *CacheFactory) ServiceAccountLister() v1.ServiceAccountLister {
	return c.sharedInformerFactory.Core().V1().ServiceAccounts().Lister()
}
