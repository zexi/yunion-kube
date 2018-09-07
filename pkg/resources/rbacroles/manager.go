package rbacroles

import (
	"yunion.io/x/yunion-kube/pkg/resources"
)

var RbacRoleManager *SRbacRoleManager

type SRbacRoleManager struct {
	*resources.SNamespaceResourceManager
}

func init() {
	RbacRoleManager = &SRbacRoleManager{
		SNamespaceResourceManager: resources.NewNamespaceResourceManager("rbacrole", "rbacroles"),
	}
}
