package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"yunion.io/yunion-kube/pkg/agent/node"
	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunion-kube/pkg/tunnelserver"
	ytypes "yunion.io/yunion-kube/pkg/types"
	"yunion.io/yunioncloud/pkg/log"
)

const (
	Token  = tunnelserver.Token
	Params = tunnelserver.Params
)

func main() {
	log.SetVerboseLevel(10)
	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func getParams() (map[string]interface{}, error) {
	return node.Params(), nil
}

func getTokenAndURL() (string, string, error) {
	token, url, err := node.TokenAndURL()
	if err != nil {
		return "", "", err
	}
	return token, url, nil
}

func isConnect() bool {
	if os.Getenv(ytypes.ENV_AGENT_CONNECT) == "true" {
		return true
	}
	_, err := os.Stat("connected")
	return err == nil
}

func connected() {
	f, err := os.Create("connected")
	if err != nil {
		f.Close()
	}
}

func cleanup(ctx context.Context) error {
	//if os.Getenv("K8S_MANAGED") != "true" {
	//return nil
	//}
	c, err := client.NewEnvClient()
	if err != nil {
		return err
	}

	args := filters.NewArgs()
	args.Add("label", fmt.Sprintf("%s=true", ytypes.AGENT_LABEL))

	containers, err := c.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return err
	}

	for _, container := range containers {
		if _, ok := container.Labels["io.kubernetes.pod.namespace"]; ok {
			continue
		}

		if strings.Contains(container.Names[0], "share-mnt") {
			continue
		}

		container := container
		go func() {
			time.Sleep(15 * time.Second)
			log.Infof("Removing unmanaged agent %s(%s)", container.Names[0], container.ID)
			c.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{Force: true})
		}()
	}

	return nil
}

func run() error {
	log.Debugf("Yunion kubernetes agent is starting")

	params, err := getParams()
	if err != nil {
		return err
	}

	bytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	log.Debugf("params: %s", string(bytes))

	token, server, err := getTokenAndURL()
	if err != nil {
		return err
	}

	headers := map[string][]string{
		Token:  {token},
		Params: {base64.StdEncoding.EncodeToString(bytes)},
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return err
	}

	onConnect := func(ctx context.Context) error {
		connectConfig := fmt.Sprintf("https://%s/connect/config", serverURL.Host)
		log.Debugf("Server connectConfig url: %q", connectConfig)
		go func() {
			log.Infof("Starting plan monitor")
			for {
				select {
				case <-time.After(2 * time.Minute):
					log.Infof("2 mins goes")
				case <-ctx.Done():
					return
				}
			}
		}()
		return nil
	}

	for {
		wsURL := fmt.Sprintf("wss://%s/connect", serverURL.Host)
		if !isConnect() {
			wsURL += "/register"
		}
		log.Infof("Connecting to %q with token %q", wsURL, token)
		remotedialer.ClientConnect(wsURL, http.Header(headers), nil, func(proto, address string) bool {
			switch proto {
			case "tcp":
				return true
			case "unix":
				return address == "/var/run/docker.sock"
			}
			return false
		}, onConnect)
		time.Sleep(5 * time.Second)
	}
}
