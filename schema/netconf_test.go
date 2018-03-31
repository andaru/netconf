package schema

import (
	"bytes"
	"context"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/andaru/netconf/rfc6242"
	"github.com/stretchr/testify/assert"
)

type mockEventHandler struct {
	input  *rfc6242.Decoder
	output *rfc6242.Encoder

	cancel context.CancelFunc

	cfgInitError error

	sawCancel        bool
	sawRPC           []xml.StartElement
	sawCaps          []string
	sawSessionID     *string
	sawInit          bool
	sawHelloEnd      bool
	sawFinished      bool
	sawRPCReply      []xml.StartElement
	sawRPCError      []xml.StartElement
	sawError         []string
	sawCriticalError []string
	sawNotification  []xml.StartElement
}

func (x *mockEventHandler) Init(ctx context.Context) error {
	x.sawInit = true
	return x.cfgInitError
}
func (x *mockEventHandler) RemoteCapabilities(ctx context.Context, caps []string) {
	for _, cap := range caps {
		x.sawCaps = append(x.sawCaps, cap)
		if cap == capBase11 {
			rfc6242.SetChunkedFraming(x.input, x.output)
		}
	}
}
func (x *mockEventHandler) RemoteSessionID(ctx context.Context, id string) {
	x.sawSessionID = &id
}
func (x *mockEventHandler) HelloReceived(ctx context.Context) {
	x.sawHelloEnd = true
}
func (x *mockEventHandler) RPC(ctx context.Context, e xml.StartElement) {
	ce := xml.CopyToken(e).(xml.StartElement)
	x.sawRPC = append(x.sawRPC, ce)
}
func (x *mockEventHandler) RPCError(ctx context.Context, e xml.StartElement) {
	ce := xml.CopyToken(e).(xml.StartElement)
	x.sawRPC = append(x.sawRPCError, ce)
}
func (x *mockEventHandler) RPCReply(ctx context.Context, e xml.StartElement) {
	ce := xml.CopyToken(e).(xml.StartElement)
	x.sawRPCReply = append(x.sawRPCReply, ce)
}
func (x *mockEventHandler) Notification(ctx context.Context, e xml.StartElement) {
	ce := xml.CopyToken(e).(xml.StartElement)
	x.sawNotification = append(x.sawNotification, ce)
}
func (x *mockEventHandler) Finished(ctx context.Context, err error) {
	x.sawFinished = true
}

// EventSender interface implementation

func (x *mockEventHandler) CriticalError(ctx context.Context, err error) {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	x.sawCriticalError = append(x.sawCriticalError, msg)
	x.cancel()
	x.sawCancel = true
}
func (x *mockEventHandler) Error(ctx context.Context, err error) {
	var msg string
	if err != nil {
		msg = err.Error()
	}
	x.sawError = append(x.sawError, msg)
}

func TestNodeNETCONF(t *testing.T) {
	for _, tc := range []struct {
		schema *NETCONFSchema
		name   string
		input  string

		wantCaps         []string
		wantRPC          []xml.StartElement
		wantRPCReply     []xml.StartElement
		wantNotification []xml.StartElement
		wantWaitErr      error
		wantCancel       bool

		wantError         []string
		wantCriticalError []string
	}{
		{
			name:   "client data in chunked-frame mode",
			schema: NewServerSchema(),
			input: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
  </capabilities>
</hello>
]]>]]>
#113
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <get></get>
</rpc>
##
`,
			wantCaps: []string{capBase10, capBase11},
			wantRPC: []xml.StartElement{
				{Name: xml.Name{Space: nsNC, Local: "get"}, Attr: []xml.Attr{}},
			},
		},

		{
			name:   "client data in end-of-message mode",
			schema: NewServerSchema(),
			input: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
  </capabilities>
</hello>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <foo></foo>
</rpc>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <bar xmlns="ns:test"></bar>
</rpc>
]]>]]>
`,
			wantCaps: []string{capBase10},
			wantRPC: []xml.StartElement{
				{
					Name: xml.Name{Space: nsNC, Local: "foo"},
					Attr: []xml.Attr{},
				},
				{
					Name: xml.Name{Space: "ns:test", Local: "bar"},
					Attr: []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: "ns:test"}},
				},
			},
		},

		{
			name:   "error:malformed-message",
			schema: NewServerSchema(),
			input: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
  </capabilities>
</hello>
]]>]]>
<?xml version="1.0" encoding="UTF-8"?>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <fooz></foo>
</rpc>
]]>]]>
`,
			wantCaps: []string{capBase10},
			wantRPC: []xml.StartElement{
				{
					Name: xml.Name{Space: nsNC, Local: "fooz"},
					Attr: []xml.Attr{},
				},
			},
			wantCancel: true,
			wantCriticalError: []string{
				"rpc error tag:malformed-message XML syntax error on line 11: element <fooz> closed by </foo>"},
		},

		{
			name:   "schema error:minoccurs:capability",
			schema: NewServerSchema(),
			input: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
  </capabilities>
</hello>
]]>]]>
`,
			wantError: []string{
				"constraint min-occurs:1 (saw 0) failed on element <capability xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">"},
		},

		{
			name:   "schema error:maxoccurs:capabilities",
			schema: NewServerSchema(),
			input: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
  </capabilities>
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.0</capability>
  </capabilities>
</hello>
]]>]]>
`,
			wantCaps: []string{capBase10, capBase10},
			wantError: []string{
				"constraint max-occurs:1 (saw 2) failed on element <capabilities xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			check := assert.New(t)
			buf := &bytes.Buffer{}
			inputStream := rfc6242.NewDecoder(strings.NewReader(tc.input))
			outputStream := rfc6242.NewEncoder(buf)
			d := xml.NewDecoder(inputStream)

			ctx, cancel := context.WithCancel(context.Background())
			evr := &mockEventHandler{input: inputStream, output: outputStream, cancel: cancel}
			p := &Parser{Schema: tc.schema, EV: evr, Cancel: cancel}

			check.NoError(p.Run(ctx, d))
			check.Equal(tc.wantWaitErr, p.Wait())
			// confirm EventReceiver interface functions are called
			check.True(evr.sawInit, "EventReceiver.Init fail")
			check.True(evr.sawFinished, "EventReceiver.Finished fail")
			check.Equal(tc.wantCancel, evr.sawCancel, "EventReceiver.Cancel fail")
			check.Equal(tc.wantCaps, evr.sawCaps, "EventReceiver.Capabilities fail")
			check.Equal(tc.wantRPC, evr.sawRPC, "EventReceiver.RPC fail")
			check.Equal(tc.wantRPCReply, evr.sawRPCReply, "EventReceiver.RPCReply fail")
			check.Equal(tc.wantNotification, evr.sawNotification, "EventReceiver.Notification fail")
			check.EqualValues(tc.wantError, evr.sawError, "EventSender.Error fail")
			check.EqualValues(tc.wantCriticalError, evr.sawCriticalError,
				"EventSender.CriticalError fail")
		})

	}
}
