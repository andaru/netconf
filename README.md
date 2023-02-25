# NETCONF support libraries for golang #

Support libraries for writing [NETCONF](https://tools.ietf.org/html/rfc6241) applications in
[Go](https://golang.org/).

## Install ##

Add to your go module, running the tests to be sure.

```bash
# add netconf package to your go module
$ go get github.com/andaru/netconf
# run the tests
$ (go test github.com/andaru/netconf/... && echo "OK: tests passed") || echo "FAIL: uhoh, tests failed"
```

## Introduction ##

With these libraries, it's easy to build NETCONF applications.

By taking care of the messy and inconvenient parts of NETCONF transport and message handling, these libraries let you write fully-featured NETCONF clients and servers with support for any underlying network transport.

## Quickstart ##

### Client ###

* Instantiate a new `session.Session` using an `io.Reader` and `io.WriteCloser` from your transport
 (client or server), typically via SSH or TLS.

```go
import (
  "log"

  "github.com/andaru/netconf/session"
)

type ClientHandler struct {
  // any local client state goes here
}

func (h *ClientHandler) OnEstablished(s *session.Session) {
  // called once by the session runner, after session start-up, framing mode and capabilities exchange
  for _, capabilityString := range s.State.Capabilties {
    // inspect the remote peer's capabilities against your own (s.Config.Capabilities)
  }
  // use the established trigger an RPC request immediately (alternatively, start an RPC request sender
  // goroutine to run alongside the client handler, using the OnClose handler method to shut it down)
  _, err := s.Outgoing().Write([]byte(`<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">...</rpc>`))
}

// OnMessage is called for each message received by the session from the peer
func (h *ClientHandler) OnMessage(s *session.Session) {
  // Consider use of the github.com/antchfx/xmlquery package instead of io.ReadAll to
  // parse NETCONF response XML data from the server (<rpc-reply> elements).

  // read the entire message (all chunks, when in chunked mode)
  _, err := io.ReadAll(s.Incoming())
  // typical read error handling
  if err != nil {
    if err != session.ErrEndOfStream {
      s.AddError(err)
      // abnormal session termination occurred (likely a transport error)
    }
    // move the status to error state, which will terminate the session and call OnError()
    s.State.Status = session.StatusError
    return
  }
  // otherwise, handle server response, then...
  // send another request (default server capabilities require <rpc> to be sent one at a time)...
}

// OnError is called when the session status changes to session.StatusError
func (h *ClientHandler) OnError(s *session.Session) { /* look at s.Errors() */ }

// OnClose is called when the session status changes to session.StatusClosed
func (h *ClientHandler) OnClose(s *session.Session) {}

func main() {
  // setup transport io.Reader as r, io.WriteCloser as w
  clientHandler := &ClientHandler{}
  // Create a NETCONF session with base and chunked capabilities
  s := session.New(r, w, session.Config{
    Capabilities: {
      "urn:ietf:params:netconf:base:1.0", // required by all clients
      "urn:ietf:params:netconf:base:1.1" // your client will support chunked encoding
    },
  })
  // Run our client handler on the session
  s.Run(clientHandler)
}
```

## Key Features ##

* NETCONF stream decoders and encoders with full NETCONF 1.1 support
* Support for both NETCONF `:base:1.0` (aka _end of message delimited_) and `:base:1.1` (aka _chunked_) framing.
* A NETCONF `session.Session` type with corresponding `session.Handler` interface.
  * Performs session initialization (`<hello>` and `<capabilities>` exchange) and session validation
    common to both client and server sessions, as well as any required framing mode switch.
  * Allows NETCONF application development based on `session.Handler` event handlers along with access
    to the session's current `Incoming` and `Outgoing` messages (as handler callback arguments).
  * Supports both client and server session customization.
  * Use `*xml.Decoder` or any other consumer supporting an `io.Reader` source to consume NETCONF messages.
  * Use `*xml.Encoder` or any other producer supporting an `io.WriteCloser` destination to produce NETCONF messages.

### Related libraries under development ###

* A streaming gRPC protocol for passing transport data
  * Allows e.g., `openssh` calling a small binary using the protocol client to call into a central NETCONF management agent running the protocol server
* A document object model (perhaps an extension of `xmlquery`)
* Encoding, decoding and validation of YANG data, including:
  * [RFC6020 XML][https://tools.ietf.org/html/rfc6020] and [RFC7951 JSON][https://tools.ietf.org/html/rfc7951] encoding and   decoding of YANG data.
