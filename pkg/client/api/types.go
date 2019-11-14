package api

import (
	apps "k8s.io/api/apps/v1beta2"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/yunion-kube/pkg/apis"
)

type ResourceName = string
type KindName = string

const (
	ResourceNameConfigMap               ResourceName = "configmaps"
	ResourceNameDaemonSet               ResourceName = "daemonsets"
	ResourceNameDeployment              ResourceName = "deployments"
	ResourceNameEvent                   ResourceName = "events"
	ResourceNameHorizontalPodAutoscaler ResourceName = "horizontalpodautoscalers"
	ResourceNameIngress                 ResourceName = "ingresses"
	ResourceNameJob                     ResourceName = "jobs"
	ResourceNameCronJob                 ResourceName = "cronjobs"
	ResourceNameNamespace               ResourceName = "namespaces"
	ResourceNameNode                    ResourceName = "nodes"
	ResourceNamePersistentVolumeClaim   ResourceName = "persistentvolumeclaims"
	ResourceNamePersistentVolume        ResourceName = "persistentvolumes"
	ResourceNamePod                     ResourceName = "pods"
	ResourceNameReplicaSet              ResourceName = "replicasets"
	ResourceNameSecret                  ResourceName = "secrets"
	ResourceNameService                 ResourceName = "services"
	ResourceNameStatefulSet             ResourceName = "statefulsets"
	ResourceNameEndpoint                ResourceName = "endpoints"
	ResourceNameStorageClass            ResourceName = "storageclasses"
	ResourceNameRole                    ResourceName = "roles"
	ResourceNameRoleBinding             ResourceName = "rolebindings"
	ResourceNameClusterRole             ResourceName = "clusterroles"
	ResourceNameClusterRoleBinding      ResourceName = "clusterrolebindings"
	ResourceNameServiceAccount          ResourceName = "serviceaccounts"
	ResourceNameLimitRange              ResourceName = "limitranges"
	ResourceNameResourceQuota           ResourceName = "resourcequotas"
)

type ResourceMap struct {
	GroupVersionResourceKind GroupVersionResourceKind
	Namespaced               bool
}

type GroupVersionResourceKind struct {
	schema.GroupVersionResource
	Kind string
}

var KindToResourceMap = map[string]ResourceMap{
	ResourceNameConfigMap: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameConfigMap,
			},
			Kind: api.KindNameConfigMap,
		},
		Namespaced: true,
	},
	ResourceNameDaemonSet: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    extensionsv1beta1.GroupName,
				Version:  extensionsv1beta1.SchemeGroupVersion.Version,
				Resource: ResourceNameDaemonSet,
			},
			Kind: api.KindNameDaemonSet,
		},
		Namespaced: true,
	},
	ResourceNameDeployment: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: ResourceNameDeployment,
			},
			Kind: api.KindNameDeployment,
		},
		Namespaced: true,
	},
	ResourceNameEvent: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameEvent,
			},
			Kind: api.KindNameEvent,
		},
		Namespaced: true,
	},

	ResourceNameHorizontalPodAutoscaler: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    autoscalingv1.GroupName,
				Version:  autoscalingv1.SchemeGroupVersion.Version,
				Resource: ResourceNameHorizontalPodAutoscaler,
			},
			Kind: api.KindNameHorizontalPodAutoscaler,
		},
		Namespaced: true,
	},
	ResourceNameIngress: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    extensionsv1beta1.GroupName,
				Version:  extensionsv1beta1.SchemeGroupVersion.Version,
				Resource: ResourceNameIngress,
			},
			Kind: api.KindNameIngress,
		},
		Namespaced: true,
	},
	ResourceNameJob: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    batchv1.GroupName,
				Version:  batchv1.SchemeGroupVersion.Version,
				Resource: ResourceNameJob,
			},
			Kind: api.KindNameJob,
		},
		Namespaced: true,
	},
	ResourceNameCronJob: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    batchv1beta1.GroupName,
				Version:  batchv1beta1.SchemeGroupVersion.Version,
				Resource: ResourceNameCronJob,
			},
			Kind: api.KindNameCronJob,
		},
		Namespaced: true,
	},
	ResourceNameNamespace: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameNamespace,
			},
			Kind: api.KindNameNamespace,
		},
		Namespaced: false,
	},
	ResourceNameNode: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameNode,
			},
			Kind: api.KindNameNode,
		},
		Namespaced: false,
	},
	ResourceNamePersistentVolumeClaim: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNamePersistentVolumeClaim,
			},
			Kind: api.KindNamePersistentVolumeClaim,
		},
		Namespaced: true,
	},
	ResourceNamePersistentVolume: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNamePersistentVolume,
			},
			Kind: api.KindNamePersistentVolume,
		},
		Namespaced: false,
	},
	ResourceNamePod: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNamePod,
			},
			Kind: api.KindNamePod,
		},
		Namespaced: true,
	},
	ResourceNameReplicaSet: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: ResourceNameReplicaSet,
			},
			Kind: api.KindNameReplicaSet,
		},
		Namespaced: true,
	},
	ResourceNameSecret: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameSecret,
			},
			Kind: api.KindNameSecret,
		},
		Namespaced: true,
	},
	ResourceNameService: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameService,
			},
			Kind: api.KindNameService,
		},
		Namespaced: true,
	},
	ResourceNameStatefulSet: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    apps.GroupName,
				Version:  apps.SchemeGroupVersion.Version,
				Resource: ResourceNameStatefulSet,
			},
			Kind: api.KindNameStatefulSet,
		},
		Namespaced: true,
	},
	ResourceNameEndpoint: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameEndpoint,
			},
			Kind: api.KindNameEndpoint,
		},
		Namespaced: true,
	},
	ResourceNameStorageClass: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    storagev1.GroupName,
				Version:  storagev1.SchemeGroupVersion.Version,
				Resource: ResourceNameStorageClass,
			},
			Kind: api.KindNameStorageClass,
		},
		Namespaced: false,
	},

	ResourceNameRole: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: ResourceNameRole,
			},
			Kind: api.KindNameRole,
		},
		Namespaced: true,
	},
	ResourceNameRoleBinding: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: ResourceNameRoleBinding,
			},
			Kind: api.KindNameRoleBinding,
		},
		Namespaced: true,
	},
	ResourceNameClusterRole: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: ResourceNameClusterRole,
			},
			Kind: api.KindNameClusterRole,
		},
		Namespaced: false,
	},
	ResourceNameClusterRoleBinding: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    rbacv1.GroupName,
				Version:  rbacv1.SchemeGroupVersion.Version,
				Resource: ResourceNameClusterRoleBinding,
			},
			Kind: api.KindNameClusterRoleBinding,
		},
		Namespaced: false,
	},
	ResourceNameServiceAccount: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameServiceAccount,
			},
			Kind: api.KindNameServiceAccount,
		},
		Namespaced: true,
	},
	ResourceNameLimitRange: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameLimitRange,
			},
			Kind: api.KindNameLimitRange,
		},
		Namespaced: true,
	},
	ResourceNameResourceQuota: {
		GroupVersionResourceKind: GroupVersionResourceKind{
			GroupVersionResource: schema.GroupVersionResource{
				Group:    corev1.GroupName,
				Version:  corev1.SchemeGroupVersion.Version,
				Resource: ResourceNameResourceQuota,
			},
			Kind: api.KindNameResourceQuota,
		},
		Namespaced: true,
	},
}

func GetResourceKinds() sets.String {
	kinds := sets.NewString()
	for keyPlural := range KindToResourceMap {
		kinds.Insert(keyPlural)
	}
	return kinds
}

func TranslateKindPlural(plural string) string {
	if GetResourceKinds().Has(plural) {
		return plural
	}
	switch plural {
	case "k8s_services":
		return ResourceNameService
	case "k8s_nodes":
		return ResourceNameNode
	case "k8s_endpoints":
		return ResourceNameEndpoint
	}
	return plural
}
