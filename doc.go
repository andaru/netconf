/*
Package netconf is a set of NETCONF (RFC6242) support libraries.

Doing the heavy lifting of protocol framing (decoding
and encoding), message layer splitting and session initialization,
these libraries allow easy NETCONF client and server application development.

NETCONF Session consume and present standard io.Reader and io.WriteCloser
interfaces via a NETCONF messages layer implementation to offer
per-message io.Reader/WriteCloser to applications, to integrate with
the golang SSH server and xml.{Decoder,Encoder}, to provide compatibility
with io.Copy, and more.

Both NETCONF 1.0 (end of message) and 1.1 (chunked framing) protocols
are supported with automatic framing mode switch based on capability exchange.

See the session sub-directory for more information about Session objects
and Handler implementations.
*/
package netconf
