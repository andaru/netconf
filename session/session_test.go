package session

import (
	"bytes"
	"strings"
	"testing"

	"github.com/antchfx/xmlquery"
	"github.com/stretchr/testify/assert"
)

func TestSessionHello(t *testing.T) {
	for _, tc := range []struct {
		config  Config
		input   string
		wantErr string
	}{
		{
			config:  Config{},
			wantErr: "end of stream",
		},
		{
			config:  Config{ID: 1},
			wantErr: "end of stream",
		},
		{
			config:  Config{ID: 1},
			input:   `<foo></foo>]]>]]>`,
			wantErr: "missing <hello> element",
		},
		{
			config: Config{ID: 1},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
	</capability>
</capabilities>
</hello>`,
			wantErr: "unexpected EOF",
		},
		{
			config: Config{},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>urn:ietf:params:netconf:base:1.1
	</capability>
</capabilities>
</hello>]]>]]>`,
			wantErr: "no session-id received for client session",
		},
		{
			config: Config{ID: 1},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>urn:ietf:params:netconf:base:1.1
	</capability>
</capabilities>
<session-id>123</session-id>
</hello>]]>]]>`,
			wantErr: "session-id received from client peer",
		},
		{
			config: Config{ID: 1},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
	</capability>
</capabilities>
</hello>
]]>]]>`,
			wantErr: "missing non-empty <capability> element(s)",
		},
		{
			config: Config{ID: 1},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
</hello>
]]>]]>`,
			wantErr: "missing non-empty <capability> element(s)",
		},
		{
			config: Config{ID: 1, Capabilities: Capabilities{capBase10}},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.1
	</capability>
</capabilities>
</hello>
]]>]]>`,
			wantErr: "session failed to negotiate framing mode",
		},
		{
			config: Config{ID: 1, Capabilities: Capabilities{capBase11}},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
</hello>
]]>]]>`,
			wantErr: "session failed to negotiate framing mode",
		},
		{
			config: Config{ID: 1, Capabilities: Capabilities{capBase11}},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.1
	</capability>
</capabilities>
</hello>
]]>]]>`,
		},
		{
			config: Config{ID: 1, Capabilities: Capabilities{capBase10}},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
</hello>
]]>]]>`,
		},
		{
			config: Config{ID: 1, Capabilities: Capabilities{capBase11, capBase10}},
			input: `
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
</hello>
]]>]]>`,
		},
		{
			config: Config{ID: 1, Capabilities: Capabilities{capBase11, capBase10}},
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
]]>]]>`,
		},
		{
			config: Config{Capabilities: Capabilities{capBase11, capBase10}},
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
<session-id>123</session-id>
</hello>
]]>]]>`,
		},
		{
			config: Config{Capabilities: Capabilities{capBase11, capBase10}},
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
<session-id>-123</session-id>
</hello>
]]>]]>`,
			wantErr: "invalid session-id value",
		},
		{
			config: Config{Capabilities: Capabilities{capBase11, capBase10}},
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
<session-id>  </session-id>
</hello>
]]>]]>`,
			wantErr: "missing session-id value",
		},
	} {
		t.Run(tc.input, func(t *testing.T) {
			src := strings.NewReader(tc.input)
			dst := closeBuffer{&bytes.Buffer{}}
			s := New(src, dst, tc.config)
			a := assert.New(t)
			if ok := s.InitialHandshake(); tc.wantErr != "" {
				if a.False(ok) {
					a.Equal(tc.wantErr, s.Errors()[0].Error())
					a.Equal(StatusError, s.State.Status)
				}
			} else {
				a.Len(s.Errors(), 0)
				a.Equal(StatusEstablished, s.State.Status)
			}
		})
	}
}

func TestSessionEstablished(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  Config
		input   string
		output  string
		wantErr string
	}{
		{
			name:   "end of message framing",
			config: Config{Capabilities: Capabilities{capBase10}},
			input: `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.1
	</capability>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
<session-id>123</session-id>
</hello>
]]>]]>
<!-- include a few empty RPCs to show we can read messages -->
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
]]>]]>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
]]>]]>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
]]>]]>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
]]>]]>`,
			output: `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities><capability>urn:ietf:params:netconf:base:1.0</capability></capabilities></hello>]]>]]><rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>]]>]]><rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>]]>]]><rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>]]>]]><rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>]]>]]>`,
		},
		{
			name:   "chunked framing",
			config: Config{Capabilities: Capabilities{capBase10, capBase11}},
			input: `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.1
	</capability>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
<session-id>123</session-id>
</hello>
]]>]]>
#59
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
#59
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
#59
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
#59
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
##
`,
			output: `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><capabilities><capability>urn:ietf:params:netconf:base:1.0</capability><capability>urn:ietf:params:netconf:base:1.1</capability></capabilities></hello>]]>]]>
#60
<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>
#60
<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>
#60
<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>
#60
<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>
##
`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := assert.New(t)
			// src is our NETCONF transport reader (from test case input string)
			src := strings.NewReader(tc.input)
			// dst is our test NETCONF transport (a simple writable buffer with a no-op Close method,
			// making it an io.WriteCloser needed for the session New() constructor)
			dst := closeBuffer{&bytes.Buffer{}}
			// create a NETCONF session for our test server implementation
			s := New(src, dst, tc.config)
			// perform the initial handshake and capabilities exchange
			if ok := s.InitialHandshake(); tc.wantErr != "" {
				if a.False(ok) {
					a.Equal(tc.wantErr, s.Errors()[0].Error())
					a.Equal(StatusError, s.State.Status)
				}
			} else {
				// ensure we have no errors and have moved to the "established" session state
				a.Len(s.Errors(), 0)
				a.Equal(StatusEstablished, s.State.Status)
			}
			// handle incoming NETCONF messages from the session until the stream ends
			var err error
			var numRPC int
			for err == nil {
				var n *xmlquery.Node
				if n, err = xmlquery.Parse(s.Incoming()); err == nil {
					a.NotNil(n)
					// confirm we are a client
					a.Equal(uint32(0), s.Config.ID)
					// confirm the session got the server's session-id
					a.Equal(uint32(123), s.State.ID)
					// handle the RPC requests from the client
					for range n.SelectElements("rpc") {
						t.Logf("node %#v", n.OutputXML(true))
						numRPC++
						s.Outgoing().Write([]byte(`<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>`))
					}
				} else {
					a.Nil(n)
				}
				a.Nil(s.Outgoing().Close())
			}
			a.Equal(4, numRPC)
			// we expect the ErrEndOfStream error to arrive on the session
			a.Equal(err, ErrEndOfStream)
			a.Equal(tc.output, dst.String())
		})
	}
}

type testHandler struct {
	t *testing.T
	a *assert.Assertions
	i int
}

func (th *testHandler) OnEstablish(s *Session) {
	th.a.Equal(StatusEstablished, s.State.Status)
}

func (th *testHandler) OnMessage(s *Session) {
	node, err := xmlquery.Parse(s.Incoming())
	if err != nil {
		// no transport errors can occur, so we will only see ErrEndOfStream once we've read all the messages
		th.a.ErrorIs(err, ErrEndOfStream)
		s.State.Status = StatusClosed
		return
	}
	// (if err is nil from xmlquery.Parse, node must be non-nil)
	th.a.NoError(err)
	th.a.NotNil(node)
	for i, rpc := range node.SelectElements("rpc") {
		th.t.Logf("rpc[%d] %s", i, rpc.OutputXML(true))
		switch th.i {
		case 0:
			th.a.NotNil(rpc.SelectElement("one"))
		case 1:
			th.a.NotNil(rpc.SelectElement("two"))
		case 2:
			th.a.NotNil(rpc.SelectElement("three"))
		case 3:
			th.a.NotNil(rpc.SelectElement("four"))
		default:
			th.a.FailNow("too many RPC requests")
		}
		th.i++
	}
}

func (th *testHandler) OnError(s *Session) {

}

func (th *testHandler) OnClose(s *Session) {

}

func TestSessionRun(t *testing.T) {
	for _, tc := range []struct {
		input  string
		config Config
	}{
		{
			input: `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
	<capability>
		urn:ietf:params:netconf:base:1.1
	</capability>
	<capability>
		urn:ietf:params:netconf:base:1.0
	</capability>
</capabilities>
<session-id>123</session-id>
</hello>
]]>]]>
#70
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><one></one></rpc>
#70
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><two></two></rpc>
#74
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><three></three></rpc>
#72
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"><four></four></rpc>
##
`,
			config: Config{Capabilities: Capabilities{capBase10, capBase11}},
		},
	} {
		t.Run(tc.input, func(t *testing.T) {
			handler := &testHandler{t: t, a: assert.New(t)}
			rdr := strings.NewReader(tc.input)
			w := closeBuffer{&bytes.Buffer{}}
			ses := New(rdr, w, tc.config)
			ses.Run(handler)
		})
	}
}

// closeBuffer just adds a no-op Close() method to a bytes.Buffer, making it
// also implement io.Closer and thus io.WriteCloser
type closeBuffer struct{ *bytes.Buffer }

func (cb closeBuffer) Close() error { return nil }
