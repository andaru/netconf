package rfc6242_test

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/andaru/netconf/rfc6242"
)

// This example shows an example NETCONF client processing the beginning of a
// session with a NETCONF server, including processing the server's <hello>
// element and the decision to upgrade to :base:1.1 or "chunked-message"
// framing protocol.
//
// See RFC6242 section 4.1 for further information about this process.
func Example_framingProtocolUpgrade() {
	// serverSessionData is mock data representing the entire contents of the
	// NETCONF server session. With a real server, the server's <hello> message
	// below occurs at the same time the client sends its own <hello> message.
	//
	// Our (implied) client then sent an <rpc> request document with message-id
	// "m-112".
	//
	// The remainder of the server session data is the response to this RPC,
	// encoded in :base:1.1 chunked-framing format because the client and
	// server both offered the :base:1.1 capability in their <hello> messages.
	serverSessionData := `<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
  </capabilities>
  <session-id>42</session-id>
</hello>
]]>]]>
#1
<
##

#100
rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="m-112">
<ok></ok>
</rpc-reply>
##
`

	// Create an io.Reader to represent the client's input stream from the
	// server. In real uses this is replaced with a network transport, such
	// as SSH or TLS. This is the raw RFC6242 input, including framing data.
	framedInputStream := strings.NewReader(serverSessionData)

	// serverTransport is the raw server output and will be assigned the SSH or
	// TLS transport writer in a real usage.
	var serverTransport io.Writer

	// Create the output stream the client will use to send to the server transport.
	outputStream := rfc6242.NewEncoder(serverTransport)
	// outputStream implements io.Writer
	var _ io.Writer = outputStream

	// Create a *rfc6242.Decoder (io.Reader and io.WriterTo) to offer the
	// decoded input stream.
	inputStream := rfc6242.NewDecoder(framedInputStream)

	// Decode XML from the input stream
	netconfNS := "urn:ietf:params:xml:ns:netconf:base:1.0"
	nameHello := xml.Name{Space: netconfNS, Local: "hello"}
	nameRPCReply := xml.Name{Space: netconfNS, Local: "rpc-reply"}
	// :base:1.1 protocol capability
	capBase11 := "urn:ietf:params:netconf:base:1.1"

	// *xml.Decoder uses the io.Reader interface of the inputStream
	var _ io.Reader = inputStream
	// Process the NETCONF server session input
	d := xml.NewDecoder(inputStream)
	for {
		token, err := d.Token()
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Token() error: %v\n", err)
			}
			break
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name {
			case nameHello: // <hello>
				type helloElement struct {
					Capabilities []string `xml:"capabilities>capability,omitempty"`
					SessionID    string   `xml:"session-id"`
				}
				hello := helloElement{}
				if err := d.DecodeElement(&hello, &token); err != nil {
					fmt.Printf("DecodeElement() error: %v\n", err)
				} else {
					fmt.Printf("saw <hello>: %v\n", hello)
					for _, capability := range hello.Capabilities {
						if capability == capBase11 {
							rfc6242.SetChunkedFraming(inputStream, outputStream)
							fmt.Println("upgraded to :base:1.1 chunked-message framing")
							break
						}
					}
				}
			case nameRPCReply: // <rpc-reply>
				fmt.Println("saw <rpc-reply>")
			}
		}
	}

	// Output:
	// saw <hello>: {[urn:ietf:params:netconf:base:1.1] 42}
	// upgraded to :base:1.1 chunked-message framing
	// saw <rpc-reply>
}
