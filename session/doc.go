/*
Package session offers a NETCONF Session implementation.

NETCONF client and server applications implement the Handler
(session event) interface, most importantly being its
OnMessage method.

The Handler interface is implemented by NETCONF client and/or
server implementations to access the NETCONF Session message layer
for I/O.

Session implementation and execution overview

Session are created using the New function, providing the
input (src) and output (dst) along with a session Config.

The Session type handles initial <hello> and <capabilities>
exchange. For this we need to know whether the Session is
considered a client or server session (as their error checking
differs). Server sessions are those which have a non-zero
Config.ID value, while Client sessions have a zero value in
this field.

The Session manages access to the underlying transport through
an abstraction of the NETCONF message layer.  Each Session has
a "current" Incoming and Outgoing message, an io.Reader and
io.WriteCloser, respectively.

Session execution

The Run function takes a base Session (as created by New) and a
a custom Handler implementation. Run calls Handler methods, as
described in Handler documentation.  When the transport layer
sees the end-of-message token on input, the current Incoming
message returns EOF to Read, and is set for renewal before the
Handler OnMessage method is called again. On the other hand,
the lifetime of Outgoing messages is controlled locally. Calling
the Outgoing message Close ends the Outgoing message and sets
it for renewal for future calls to Outgoing (i.e., in the next
OnMessage handler).

Because the Incoming message returns EOF on each message, it
effectively masks EOF as a signal for the end of the underlying
Session input stream. To signal end of stream, an Incoming message
Read function will return ErrEndOfStream instead of EOF in case
of EOF of the underlying stream (on the first call to each Incoming)
message, meaning an unexpected EOF mid-message will not report
ErrEndOfStream until a Read to the next Incoming message is made.
*/
package session
