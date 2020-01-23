package session

import (
	"bytes"
	"strings"
	"testing"

	"github.com/andaru/netconf/message"
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
					for range n.SelectElements("rpc") {
						numRPC++
						s.Outgoing().Write([]byte(`<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"/>`))
					}
				} else {
					a.Nil(n)
				}
				a.Nil(s.Outgoing().Close())
			}
			a.Equal(4, numRPC)
			a.EqualError(err, message.ErrEndOfStream.Error())
			a.Equal(tc.output, dst.String())
		})
	}
}

type closeBuffer struct{ *bytes.Buffer }

func (cb closeBuffer) Close() error { return nil }
