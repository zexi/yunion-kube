package apis

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
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

// PortMapping is a specification of port mapping for an application deployment.
type PortMapping struct {
	// Port that will be exposed on the service.
	Port int32 `json:"port"`

	// Docker image path for the application.
	TargetPort int32 `json:"targetPort"`

	// IP protocol for the mapping, e.g., "TCP" or "UDP".
	Protocol v1.Protocol `json:"protocol"`
}

func GenerateName(base string) string {
	maxNameLength := 63
	randomLength := 5
	maxGeneratedNameLength := maxNameLength - randomLength
	if len(base) > maxGeneratedNameLength {
		base = base[:maxGeneratedNameLength]
	}
	return fmt.Sprintf("%s%s", base, rand.String(randomLength))
}

func GeneratePortMappingName(portMapping PortMapping) string {
	return GenerateName(fmt.Sprintf("%s-%d-%d-", strings.ToLower(string(portMapping.Protocol)),
		portMapping.Port, portMapping.TargetPort))
}

func (p PortMapping) ToServicePort() v1.ServicePort {
	return v1.ServicePort{
		Protocol: p.Protocol,
		Port:     p.Port,
		Name:     GeneratePortMappingName(p),
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: p.TargetPort,
		},
	}
}

type ServiceCreateOption struct {
	Type                string            `json:"type"`
	IsExternal          bool              `json:"isExternal"`
	PortMappings        []PortMapping     `json:"portMappings"`
	Selector            map[string]string `json:"selector"`
	LoadBalancerNetwork string            `json:"loadBalancerNetwork"`
}

type ServiceCreateInput struct {
	K8sNamespaceResourceCreateInput
	ServiceCreateOption
}
