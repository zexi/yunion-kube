package ssh

import (
	"encoding/base64"
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/util/ssh"
	"yunion.io/x/pkg/util/seclib"
)

// RemoteSSHBashScript executes command on remote machine
func RemoteSSHBashScript(host string, port int, username string, privateKey, content string) (string, error) {
	cli, err := ssh.NewClient(host, port, username, "", privateKey)
	if err != nil {
		return "", err
	}
	content = base64.StdEncoding.EncodeToString([]byte(content))
	tmpFile := fmt.Sprintf("/tmp/script-%s", seclib.RandomPassword(8))
	writeScript := fmt.Sprintf("echo '%s' | base64 -d > %s", content, tmpFile)
	execScript := fmt.Sprintf("bash %s", tmpFile)
	rmScript := fmt.Sprintf("rm %s", tmpFile)
	ret, err := cli.RawRun(writeScript, execScript, rmScript)
	if err != nil {
		return "", err
	}
	return strings.Join(ret, "\n"), nil
}

func RemoteSSHCommand(host string, port int, username string, privateKey, cmd string) (string, error) {
	cli, err := ssh.NewClient(host, port, username, "", privateKey)
	if err != nil {
		return "", err
	}
	ret, err := cli.RawRun(cmd)
	if err != nil {
		return "", err
	}
	return strings.Join(ret, "\n"), nil
}
