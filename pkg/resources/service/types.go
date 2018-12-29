package service

import (
	"fmt"
	"strings"

	api "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"yunion.io/x/yunion-kube/pkg/resources/common"
)

// PortMapping is a specification of port mapping for an application deployment.
type PortMapping struct {
	// Port that will be exposed on the service.
	Port int32 `json:"port"`

	// Docker image path for the application.
	TargetPort int32 `json:"targetPort"`

	// IP protocol for the mapping, e.g., "TCP" or "UDP".
	Protocol api.Protocol `json:"protocol"`
}

func GeneratePortMappingName(portMapping PortMapping) string {
	return common.GenerateName(fmt.Sprintf("%s-%d-%d-", strings.ToLower(string(portMapping.Protocol)),
		portMapping.Port, portMapping.TargetPort))
}

func (p PortMapping) ToServicePort() api.ServicePort {
	return api.ServicePort{
		Protocol: p.Protocol,
		Port:     p.Port,
		Name:     GeneratePortMappingName(p),
		TargetPort: intstr.IntOrString{
			Type:   intstr.Int,
			IntVal: p.TargetPort,
		},
	}
}
