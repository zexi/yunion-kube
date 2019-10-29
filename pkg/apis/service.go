package apis

import (
	"k8s.io/api/core/v1"
)

type Service struct {
	ObjectMeta
	TypeMeta

	// InternalEndpoint of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	InternalEndpoint Endpoint `json:"internalEndpoint"`

	// ExternalEndpoints of all kubernetes services that have the same label selector as connected Replication
	// Controller. Endpoint is DNS name merged with ports
	ExternalEndpoints []Endpoint `json:"externalEndpoints"`

	// Label selector of the service
	Selector map[string]string `json:"selector"`

	// Type determines how the service will be exposed. Valid options: ClusterIP, NodePort, LoadBalancer
	Type v1.ServiceType `json:"type"`

	// ClusterIP is usually assigned by the master. Valid values are None, empty string (""), or
	// a valid IP address. None can be specified for headless services when proxying is not required
	ClusterIP string `json:"clusterIP"`
}

type ServiceDetail struct {
	Service

	// List of Endpoint obj. that are endpoints of this Service.
	EndpointList []EndpointDetail `json:"endpointList"`

	// Type determines how the service will be exposed.  Valid options: ClusterIP, NodePort, LoadBalancer
	Type v1.ServiceType `json:"type"`

	// ClusterIP is usually assigned by the master. Valid values are None, empty string (""), or
	// a valid IP address. None can be specified for headless services when proxying is not required
	ClusterIP string `json:"clusterIP"`

	// List of events related to this Service
	EventList []Event `json:"events"`

	// PodList represents list of pods targeted by same label selector as this service.
	PodList []Pod `json:"pods"`

	// Show the value of the SessionAffinity of the Service.
	SessionAffinity v1.ServiceAffinity `json:"sessionAffinity"`
}
