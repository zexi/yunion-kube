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

	"github.com/urfave/cli"

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

func appFlags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:  "debug,d",
			Usage: "Debug logging",
		},
		cli.StringFlag{
			Name:        "node-name",
			Usage:       "Requested Hostname",
			Destination: &node.RequestedHostname,
		},
		cli.StringFlag{
			Name:        "address",
			Usage:       "IP address",
			Destination: &node.Address,
		},
		cli.StringFlag{
			Name:        "internal-address",
			Usage:       "Internal IP address",
			Destination: &node.InternalAddress,
		},
		cli.StringFlag{
			Name:        "server",
			Usage:       "Yunion kube server address",
			Value:       "https://127.0.0.1:8443",
			Destination: &node.ServerAddress,
		},
		cli.StringFlag{
			Name:        "token",
			Usage:       "Agent token for register",
			Destination: &node.AgentToken,
		},
		cli.StringFlag{
			Name:        "id",
			Usage:       "Node id for register",
			Destination: &node.NodeId,
		},
	}
}

func setupApp() *cli.App {
	app := cli.NewApp()
	app.Name = "kube-agent"
	app.Version = "0.0.1"
	app.Usage = "Yunion kubernetes agent"
	app.Before = func(ctx *cli.Context) error {
		if ctx.Bool("debug") {
			log.SetLogLevelByString(log.Logger(), "debug")
			log.SetVerboseLevel(10)
		}
		return nil
	}
	app.Author = "Yunion Technology @ 2018"
	app.Email = "lizexi@yunion.io"
	app.Flags = appFlags()
	app.Action = run
	return app
}

func main() {
	app := setupApp()

	err := app.Run(os.Args)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func run(c *cli.Context) error {
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

func getParams() (map[string]interface{}, error) {
	return node.Params()
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
