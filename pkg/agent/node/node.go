package node

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/docker/docker/client"

	"yunion.io/x/log"
)

var (
	RequestedHostname string
	Address           string
	InternalAddress   string
	AgentToken        string
	NodeId            string
	ServerAddress     string
)

func TokenAndURL() (string, string, error) {
	if len(AgentToken) == 0 {
		return "", "", fmt.Errorf("Empty AgentToken")
	}
	if len(ServerAddress) == 0 {
		return "", "", fmt.Errorf("Empty ServerAddress")
	}
	return AgentToken, ServerAddress, nil
}

func GetLocalIPAddr() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

func Params() (map[string]interface{}, error) {
	var err error
	if Address == "" {
		Address, err = GetLocalIPAddr()
		if err != nil {
			return nil, err
		}
	}
	if RequestedHostname == "" {
		RequestedHostname, _ = os.Hostname()
	}
	if NodeId == "" {
		return nil, fmt.Errorf("Node Id not specified")
	}
	params := map[string]interface{}{
		"id":                NodeId,
		"address":           Address,
		"internalAddress":   InternalAddress,
		"requestedHostname": RequestedHostname,
	}

	for k, v := range params {
		if m, ok := v.(map[string]string); ok {
			for k, v := range m {
				log.Infof("Option %s=%s", k, v)
			}
		} else {
			log.Infof("Option %s=%v", k, v)
		}
	}

	dclient, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	info, err := dclient.Info(context.Background())
	if err == nil {
		params["dockerInfo"] = info
	}

	return map[string]interface{}{
		"node": params,
	}, nil
}

func split(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 1 && result[0] == "" {
		return nil
	}
	return result
}
