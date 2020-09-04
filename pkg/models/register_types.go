package models

import (
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/yunion-kube/pkg/api"
	gotypesutil "yunion.io/x/yunion-kube/pkg/utils/gotypes"
)

func init() {
	RegisterSerializable(
		// for role bindings
		new(api.RoleRef), new(api.Subjects),
		// for federated namespace
		new(api.FederatedNamespaceSpec),
		// for federated role
		new(api.FederatedRoleSpec),
		// for federated rolebinding
		new(api.FederatedRoleBindingSpec),
		// for federated cluserrole
		new(api.FederatedClusterRoleSpec),
		// for federated clusterrolebinding
		new(api.FederatedClusterRoleBindingSpec),
	)
}

func RegisterSerializable(objs ...gotypes.ISerializable) {
	gotypesutil.RegisterSerializable(objs...)
}
