package ssh

import (
	"fmt"
	"os/exec"
	"strings"

	"yunion.io/x/log"
)

// RemoteSSHBashScript executes command on remote machine
func RemoteSSHBashScript(user, ip, password, cmd string) (string, error) {
	c := exec.Command("sshpass", "-p", password, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-q", user+"@"+ip, "bash", "-c", cmd)
	log.Infof("cmd args %s", c.Args)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error: %v, output: %s", err, string(out))
	}
	result := strings.TrimSpace(string(out))
	return result, nil
}

func RemoteSSHCommand(user, ip, password, cmd string) (string, error) {
	c := exec.Command("sshpass", "-p", password, "ssh", "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-q", user+"@"+ip, cmd)
	log.Infof("cmd args %s", c.Args)
	out, err := c.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error: %v, output: %s", err, string(out))
	}
	result := strings.TrimSpace(string(out))
	return result, nil
}
