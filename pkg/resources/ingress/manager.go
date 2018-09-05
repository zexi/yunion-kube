package ingress

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var IngressManager *SIngressManager

type SIngressManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	IngressManager = &SIngressManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("ingress", "ingresses"),
	}
}
