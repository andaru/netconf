# NETCONF support libraries for golang #

Support libraries for writing [NETCONF](https://tools.ietf.org/html/rfc6241) applications in
[Go](https://golang.org/).

## Install or update

Install the libraries, running the tests to be sure.

```bash
$ go get -u github.com/andaru/netconf/...

$ (go test github.com/andaru/netconf/... && echo "OK: tests passed") || echo "FAIL: uhoh, tests failed"
```

[Go modules](https://github.com/golang/go/wiki/Modules) are not yet supported. Please submit a pull request if you need this support very soon, otherwise I will be adding the module configuration prior to feature completeness.

## Introduction

With these libraries, it's easy to build NETCONF applications including:
  * NETCONF system/network management agents, managing requests on a datastore mapped to the local system's configuration
  * NETCONF to network-element CLI or other protocol proxies for non-NETCONF devices, e.g.,
    * OpenConfig to NETCONF proxy
    * NETCONF to oldschool CLI proxy
  * NETCONF API servers and command-line tools for network operator use

By taking care of the messy and inconvenient parts of NETCONF transport and message handling, these libraries offer a reliable
and accessible way to write fully-featured NETCONF applications with support for a wide variety of underlying transports.

## Key Features

  * NETCONF transport decoders and encoders with full NETCONF 1.1 support
    * Transport connections via standard `io.Reader` and `io.WriteCloser` interfaces, e.g.,:
      * `golang.org/x/crypto/ssh` integration for client and server sessions (provided)
      * OpenSSHd and similar `sshd` integration via gRPC transport stream service (provided)
    * Support for both NETCONF `:base:1.0` (aka _end of message delimited_) and `:base:1.1` (aka _chunked_) framing.
    * Allows switching from end-of-message to chunked framing on a live transport, as required by spec.
      * Used by the built-in `Session` implementation to automatically switch framing mode if required at the
        appropriate time, with clients having seamless access to `Read`s.

  * A base NETCONF `session.Session` type with corresponding `session.Handler` interface
    * A simple abstraction supports customization for both client and server session use.
    * Provides session initialization (`<hello>` and `<capabilities>` exchange) and session validation
      common to both client and server sessions, as well as any required framing mode switch.
    * Allows NETCONF application development based on `Session` event handlers along with access
      to the session's current `Incoming` and `Outgoing` messages.
    * Use `*xml.Decoder` or any other consumer supporting an `io.Reader` source to consume NETCONF messages.
    * Use `*xml.Encoder` or any other producer supporting an `io.WriteCloser` destination to produce NETCONF messages.
    * More complex client and server applications are built on top of the session and handler state-machine interface.

  * A gRPC based, bi-directional (aka _duplex_) transport stream service.
    * Connect to clients via an OpenSSH server subsystem command as a gRPC client to the stream service.
    * Offers access to the stream as an `io.ReadWriteCloser`.
    * Supports multiple simultaneous stream (e.g., to allow separation of `stdout` and `stderr`).
    * Due to [this issue](https://github.com/golang/go/issues/35025), the "standard" golang `ssh` package cannot be used to write NETCONF SSH servers, but the above gRPC approach with OpenSSH is used successfully in production.

### Related non-NETCONF libraries under development (in other respoisotires)

  * A document object model (perhaps an extension of `xmlquery`)
  * Encoding, decoding and validation of YANG data, including:
    * [RFC6020 XML][https://tools.ietf.org/html/rfc6020] and [RFC7951 JSON][https://tools.ietf.org/html/rfc7951] encoding and   decoding of YANG data.
