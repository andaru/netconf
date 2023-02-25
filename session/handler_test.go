package session

import (
	"bytes"
	"io"
	"math/rand"
	"strings"
	"testing"

	"github.com/andaru/netconf/message"
	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"
)

func TestSessionHandler(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		cfg   Config
	}{
		{
			name: `end of message`,
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
	<capabilities>
		<capability>
			urn:ietf:params:netconf:base:1.1
		</capability>
		<capability>
			urn:ietf:params:netconf:base:1.0
		</capability>
	</capabilities>
</hello>
]]>]]>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
</rpc>
]]>]]>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := strings.NewReader(tc.input)
			dst := closeBuffer{&bytes.Buffer{}}
			session := New(src, dst, tc.cfg)
			// our mock server handler for this one session
			handler := &mockSession{}
			a := assert.New(t)
			a.False(handler.c)
			a.False(handler.csc)
			session.Run(handler)
			a.True(handler.c)
			a.True(handler.csc)
		})
	}

	serverInput := `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.1
	</capability>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
</hello>
]]>]]>`
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
	serverHandler := &mockSession{}
	serverSession := New(serverSrc, serverDst, serverConfig)
	serverSession.Run(serverHandler)
	a := assert.New(t)
	if errs := serverSession.Errors(); a.NotEqual(nil, errs) {
		for _, err := range errs {
			t.Logf("err=%v", err)
		}
	}
}

type mockSession struct {
	docRcvd []*xmlquery.Node
	errs    []error
	c       bool
	csc     bool
	ec      bool
}

func (srv *mockSession) OnEstablish(s *Session) {
	if srv.ec {
		panic("OnEstablish must only be called once")
	}
	srv.ec = true
}
func (srv *mockSession) OnMessage(s *Session) {
	// s.Incoming provides the incoming read handle (an io.Reader implementation).
	//
	// Use xmlquery.Parse to read and parse a node structure from the
	// Incoming document's XML.  In this example, we are implementing a server,
	// so the Incoming stream document represents client requests.
	node, err := xmlquery.Parse(s.Incoming())
	switch {
	case err == message.ErrEndOfStream:
		// you must handle this error when reading s.Incoming, by setting the session status to
		// closed
		s.State.Status = StatusClosed
		return
	case err != nil:
		s.AddError(err)
		s.State.Status = StatusError
		return
	}
	root := xmlquery.CreateXPathNavigator(node)
	if root == nil {
		s.AddError(io.ErrUnexpectedEOF)
		s.State.Status = StatusError
		return
	}
	for n := root.Current().FirstChild; n != nil; n = n.NextSibling {
		srv.docRcvd = append(srv.docRcvd, n)
	}
}
func (srv *mockSession) OnError(s *Session) { srv.errs = append(srv.errs, s.Errors()...) }
func (srv *mockSession) OnClose(s *Session) { srv.c, srv.csc = true, s.State.Status == StatusClosed }

var _ Handler = &mockSession{}
