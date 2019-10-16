package compute

import (
	"fmt"

	"github.com/pkg/errors"
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/util/secrules"
)

type SSecgroupRuleCreateInput struct {
	apis.Meta

	Priority    int
	Protocol    string
	Ports       string
	PortStart   int
	PortEnd     int
	Direction   string
	CIDR        string
	Action      string
	Description string
	SecgroupId  string
}

func (input *SSecgroupRuleCreateInput) Check() error {
	rule := secrules.SecurityRule{
		Priority:  input.Priority,
		Direction: secrules.TSecurityRuleDirection(input.Direction),
		Action:    secrules.TSecurityRuleAction(input.Action),
		Protocol:  input.Protocol,
		PortStart: input.PortStart,
		PortEnd:   input.PortEnd,
		Ports:     []int{},
	}

	if len(input.Ports) > 0 {
		err := rule.ParsePorts(input.Ports)
		if err != nil {
			return errors.Wrapf(err, "ParsePorts(%s)", input.Ports)
		}
	}

	if len(input.CIDR) > 0 {
		if !regutils.MatchCIDR(input.CIDR) && !regutils.MatchIPAddr(input.CIDR) {
			return fmt.Errorf("invalid ip address: %s", input.CIDR)
		}
	} else {
		input.CIDR = "0.0.0.0/0"
	}

	if rule.PortStart > 0 && rule.PortEnd > 0 {
		if rule.PortStart != rule.PortEnd {
			input.Ports = fmt.Sprintf("%d-%d", rule.PortStart, rule.PortEnd)
		} else {
			input.Ports = fmt.Sprintf("%d", input.PortStart)
		}
	}

	return rule.ValidateRule()
}

type SSecgroupCreateInput struct {
	apis.Meta

	Name        string
	Description string
	Rules       []SSecgroupRuleCreateInput
}
