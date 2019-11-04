package apis

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectMeta is metadata about an instance of a resource.
type ObjectMeta struct {
	// kubernetes object meta
	metav1.ObjectMeta
	// onecloud cluster meta info
	*ClusterMeta
}

type TypeMeta struct {
	metav1.TypeMeta
}

type ClusterMeta struct {
	// Onecloud cluster data
	Cluster   string `json:"cluster"`
	ClusterId string `json:"clusterID"`
}

func (m ObjectMeta) GetName() string {
	return m.Name
}

func NewClusterMeta(cluster ICluster) *ClusterMeta {
	return &ClusterMeta{
		Cluster:   cluster.GetName(),
		ClusterId: cluster.GetId(),
	}
}

type ICluster interface {
	GetId() string
	GetName() string
}

// NewObjectMeta returns internal endpoint name for the given service properties, e.g.,
// NewObjectMeta creates a new instance of ObjectMeta struct based on K8s object meta.
func NewObjectMeta(k8SObjectMeta metav1.ObjectMeta, cluster ICluster) ObjectMeta {
	return ObjectMeta{
		ObjectMeta:  k8SObjectMeta,
		ClusterMeta: NewClusterMeta(cluster),
	}
}

func NewTypeMeta(typeMeta metav1.TypeMeta) TypeMeta {
	return TypeMeta{typeMeta}
}

// ResourceKind is an unique name for each resource. It can used for API discovery and generic
// code that does things based on the kind. For example, there may be a generic "deleter"
// that based on resource kind, name and namespace deletes it.
type ResourceKind string

type KindName = string

const (
	KindNameConfigMap               KindName = "ConfigMap"
	KindNameDaemonSet               KindName = "DaemonSet"
	KindNameDeployment              KindName = "Deployment"
	KindNameEvent                   KindName = "Event"
	KindNameHorizontalPodAutoscaler KindName = "HorizontalPodAutoscaler"
	KindNameIngress                 KindName = "Ingress"
	KindNameJob                     KindName = "Job"
	KindNameCronJob                 KindName = "CronJob"
	KindNameNamespace               KindName = "Namespace"
	KindNameNode                    KindName = "Node"
	KindNamePersistentVolumeClaim   KindName = "PersistentVolumeClaim"
	KindNamePersistentVolume        KindName = "PersistentVolume"
	KindNamePod                     KindName = "Pod"
	KindNameReplicaSet              KindName = "ReplicaSet"
	KindNameSecret                  KindName = "Secret"
	KindNameService                 KindName = "Service"
	KindNameStatefulSet             KindName = "StatefulSet"
	KindNameEndpoint                KindName = "Endpoints"
	KindNameStorageClass            KindName = "StorageClass"
	KindNameRole                    KindName = "Role"
	KindNameRoleBinding             KindName = "RoleBinding"
	KindNameClusterRole             KindName = "ClusterRole"
	KindNameClusterRoleBinding      KindName = "ClusterRoleBinding"
	KindNameServiceAccount          KindName = "ServiceAccount"
)
