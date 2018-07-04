package remotedialer

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"yunion.io/yunioncloud/pkg/log"
)

var (
	errFailedAuth       = errors.New("Failed authentication")
	errWrongMessageType = errors.New("Wrong websocket message type")
)

type Authorizer func(req *http.Request) (clientKey string, authed bool, err error)
type ErrorWriter func(rw http.ResponseWriter, req *http.Request, code int, err error)

type Server struct {
	ready       func() bool
	authorizer  Authorizer
	errorWriter ErrorWriter
	sessions    *sessionManager
}

func New(auth Authorizer, errorWriter ErrorWriter, ready func() bool) *Server {
	return &Server{
		ready:       ready,
		authorizer:  auth,
		errorWriter: errorWriter,
		sessions:    newSessionManager(),
	}
}

func (s *Server) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !s.ready() {
		s.errorWriter(rw, req, 503, errors.New("tunnel server not active"))
		return
	}

	clientKey, authed, err := s.authorizer(req)
	if err != nil {
		s.errorWriter(rw, req, 400, err)
		return
	}
	if !authed {
		s.errorWriter(rw, req, 401, errFailedAuth)
		return
	}

	log.Infof("Handing backend connection request [%s]", clientKey)

	upgrader := websocket.Upgrader{
		HandshakeTimeout: 5 * time.Second,
		CheckOrigin:      func(r *http.Request) bool { return true },
		Error:            s.errorWriter,
	}

	wsConn, err := upgrader.Upgrade(rw, req, nil)
	if err != nil {
		s.errorWriter(rw, req, 400, fmt.Errorf("Error during upgrade for host [%v]", clientKey))
		return
	}

	session := s.sessions.add(clientKey, wsConn)
	defer s.sessions.remove(session)

	// Don't need to associate req.Context() to the session, it will cacel otherwise
	code, err := session.serve()
	if err != nil {
		// Hijacked so we can't wirte to the client
		log.Debugf("error in remotedialer server [%d]: %v", code, err)
	}
}
