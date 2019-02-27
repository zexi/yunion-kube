package client

import (
	"fmt"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	cloudmod "yunion.io/x/onecloud/pkg/mcclient/modules"
)

func GetCloudSSHPrivateKey(session *mcclient.ClientSession) (string, error) {
	query := jsonutils.NewDict()
	query.Add(jsonutils.JSONTrue, "admin")
	ret, err := cloudmod.Sshkeypairs.List(session, query)
	if err != nil {
		return "", fmt.Errorf("Get admin keypair: %v", err)
	}
	if len(ret.Data) == 0 {
		return "", fmt.Errorf("Not found admin ssh keypair: %v", err)
	}
	keys := ret.Data[0]
	privateKey, err := keys.GetString("private_key")
	if err != nil {
		return "", fmt.Errorf("Get private_key: %v", err)
	}
	return privateKey, err
}
