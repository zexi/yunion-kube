package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"yunion.io/yunion-kube/pkg/remotedialer"
	"yunion.io/yunioncloud/pkg/log"
)

var (
	addr  string
	id    string
	debug bool
)

func main() {
	flag.StringVar(&addr, "connect", "wss://localhost:8443/connect", "Address to connect to")
	flag.StringVar(&id, "id", "foo", "Client ID")
	flag.Parse()

	if err := run(); err != nil {
		log.Fatalf("%v", err)
	}
}

func run() error {
	log.Debugf("Yunion kubernetes agent is starting")

	serverURL, err := url.Parse(addr)
	if err != nil {
		return err
	}

	onConnect := func(ctx context.Context) error {
		connectConfig := fmt.Sprintf("https://%s/connect/config", serverURL.Host)
		log.Debugf("Server connectConfig url: %q", connectConfig)
		return nil
	}

	headers := http.Header{
		"X-Tunnel-ID": []string{id},
	}

	for {
		wsURL := fmt.Sprintf("wss://%s/connect", serverURL.Host)
		log.Debugf("==url: %s", wsURL)
		remotedialer.ClientConnect(wsURL, headers, nil, func(proto, address string) bool {
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
