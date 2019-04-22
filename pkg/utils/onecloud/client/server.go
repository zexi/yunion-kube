package client

import (
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ServerHelper struct {
	*ResourceHelper
}

func NewServerHelper(s *mcclient.ClientSession) *ServerHelper {
	return &ServerHelper{
		ResourceHelper: NewResourceHelper(s, &modules.Servers),
	}
}

func (h *ServerHelper) continueWait(status string) bool {
	if strings.HasSuffix(status, "_fail") || strings.HasSuffix(status, "_failed") {
		return false
	}
	return true
}

func (h *ServerHelper) WaitRunning(id string) error {
	return h.WaitObjectStatus(id, api.VM_RUNNING, h.continueWait)
}

func (h *ServerHelper) WaitDelete(id string) error {
	return h.WaitObjectDelete(id, h.continueWait)
}
