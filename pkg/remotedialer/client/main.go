package main

import (
	"flag"
	"net/http"

	"yunion.io/yunion-kube/pkg/remotedialer"
)

var (
	addr  string
	id    string
	debug bool
)

func main() {
	flag.StringVar(&addr, "connect", "ws://localhost:8123/connect", "Address to connect to")
	flag.StringVar(&id, "id", "foo", "Client ID")
	flag.Parse()

	headers := http.Header{
		"X-Tunnel-ID": []string{id},
	}

	remotedialer.ClientConnect(addr, headers, nil, nil, nil)
}
