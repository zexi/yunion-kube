package remotedialer

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"yunion.io/x/log"
)

type ConnectAuthorizer func(proto, address string) bool

func ClientConnect(wsURL string, headers http.Header, dialer *websocket.Dialer, auth ConnectAuthorizer, onConnect func(context.Context) error) {
	if err := connectToProxy(wsURL, headers, auth, dialer, onConnect); err != nil {
		log.Errorf("Failed to connect to proxy: %v", err)
		time.Sleep(time.Duration(5) * time.Second)
	}
}

func connectToProxy(proxyURL string, headers http.Header, auth ConnectAuthorizer, dialer *websocket.Dialer, onConnect func(context.Context) error) error {
	log.Infof("Connecting to proxy url: %q", proxyURL)

	if dialer == nil {
		dialer = &websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	ws, _, err := dialer.Dial(proxyURL, headers)
	if err != nil {
		log.Errorf("Failed to connect to proxy: %v", err)
		return err
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if onConnect != nil {
		if err := onConnect(ctx); err != nil {
			return err
		}
	}

	session := newClientSession(auth, ws)
	_, err = session.serve()
	session.Close()
	return err
}
