package client

import (
	"strings"

	"github.com/pkg/errors"

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

func (h *ServerHelper) Servers() *modules.ServerManager {
	return h.ResourceHelper.Manager.(*modules.ServerManager)
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

type ServerLoginInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *ServerHelper) GetLoginInfo(id string) (*ServerLoginInfo, error) {
	ret, err := h.Servers().GetLoginInfo(h.session, id, nil)
	if err != nil {
		return nil, err
	}
	info := new(ServerLoginInfo)
	if err := ret.Unmarshal(info); err != nil {
		return nil, err
	}
	if len(info.Username) == 0 || len(info.Password) == 0 {
		return nil, errors.New("Empty username or password")
	}
	return info, nil
}
