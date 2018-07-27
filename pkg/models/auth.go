package models

import (
	"fmt"

	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/auth"

	"yunion.io/yunion-kube/pkg/options"
)

const (
	KUBE_SERVER_SERVICE    = "k8s"
	INTERNAL_ENDPOINT_TYPE = "internalURL"
)

func GetAdminSession() (*mcclient.ClientSession, error) {
	session := auth.AdminSession(options.Options.Region, "", "", "")
	if session == nil {
		return nil, fmt.Errorf("Can't get cloud session, maybe not init auth package ???")
	}
	return session, nil
}

func GetAdminCred() mcclient.TokenCredential {
	return auth.AdminCredential()
}
