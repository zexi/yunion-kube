package models

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"

	"yunion.io/x/yunion-kube/pkg/options"
)

const (
	KUBE_SERVER_SERVICE    = "k8s"
	INTERNAL_ENDPOINT_TYPE = "internalURL"
)

func GetAdminSession() (*mcclient.ClientSession, error) {
	session := auth.AdminSession(context.TODO(), options.Options.Region, "", "", "v2")
	if session == nil {
		return nil, fmt.Errorf("Can't get cloud session, maybe not init auth package ???")
	}
	return session, nil
}

func GetAdminCred() mcclient.TokenCredential {
	return auth.AdminCredential()
}
