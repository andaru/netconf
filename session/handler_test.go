package session

import "testing"

import "strings"

import "bytes"

import "math/rand"

func TestSessionHandler(t *testing.T) {
	serverInput := ""
	serverSrc := strings.NewReader(serverInput)
	serverDst := closeBuffer{&bytes.Buffer{}}
	sessionID := rand.Uint32()
	serverConfig := Config{
		ID: sessionID,
		Capabilities: Capabilities{
			"urn:ietf:params:netconf:base:1.0",
			"urn:ietf:params:netconf:base:1.1",
		},
	}
	// a minimal server implementation
	serverHandler := &minimalServer{}
	serverSession := New(serverSrc, serverDst, serverConfig)
	serverSession.Run(serverHandler)
}

type minimalServer struct{}

func (srv minimalServer) OnEstablish(s *Session) {}
func (srv minimalServer) OnMessage(s *Session)   {}
func (srv minimalServer) OnError(s *Session)     {}
func (srv minimalServer) OnClose(s *Session)     {}

var _ Handler = minimalServer{}
