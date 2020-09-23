package policy

import (
	common_policy "yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/util/rbacutils"

	"yunion.io/x/yunion-kube/pkg/api/constants"
)

const (
	ActionGet     = common_policy.PolicyActionGet
	ActionList    = common_policy.PolicyActionList
	ActionCreate  = common_policy.PolicyActionCreate
	ActionUpdate  = common_policy.PolicyActionUpdate
	ActionDelete  = common_policy.PolicyActionDelete
	ActionPerform = common_policy.PolicyActionPerform
)

var (
	preDefinedDefaultPolicies = []rbacutils.SRbacPolicy{
		{
			Auth:  true,
			Scope: rbacutils.ScopeDomain,
			Rules: []rbacutils.SRbacRule{
				{
					Service: constants.ServiceType,
					// Resource: "releases",
					// Action:   rbacutils.WILD_MATCH,
					Result: rbacutils.Allow,
				},
			},
		},
	}
)

func init() {
	common_policy.AppendDefaultPolicies(preDefinedDefaultPolicies)
}
