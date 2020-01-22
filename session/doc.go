/*
Package session provides NETCONF Session and their Handler.

NETCONF client and server applications implement the Handler
(session event) interface, most important being its
OnMessage method. Handler access uncooked transport I/O
via the Session's Incoming and Outgoing messages.

Session implementation and execution overview

Session for either client or server use are created using
the New function, providing the input (src) and output (dst)
along with a session Config.  The config has two fields, at
least one of which (Capabilities) must be populated for a
session to establish successfully, so that protocol framing
capability exchange can occur. See the sections on the
Session.Config fields below.

The Session manages access to the underlying transport through
an abstraction of the NETCONF message layer.  Each Session has
a "current" Incoming and Outgoing message, an io.Reader and
io.WriteCloser, respectively, which applications use to read
from and write to the session's current message.

Session.Config.ID (uint32) field

The Session type handles initial <hello> and <capabilities>
exchange (during the call to InitialHandshake).
For this we need to know whether the Session is
considered a client or server session (as their error checking
differs). Server sessions are those which have a non-zero
Config.ID value, while Client sessions have a zero value in
this field.

For example, a server's "session manager" would
provide the Config.ID value when creating the session,
while clients ignore Config.ID and can retrieve the session-id via
the State.ID field as required, e.g., when calling the
<kill-session> RPC.

Session.Config.Capabilities (string slice) field

This slice field must be populated to define the <capability> URLs
which will be sent to the peer during session initialization
and capability exchange.

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
Session input stream. To signal end of stream, the first call (only)
to an Incoming message's Read function will return ErrEndOfStream
instead of EOF in case of EOF of the underlying stream (on the
first call to each Incoming) message, meaning an unexpected EOF
mid-message will not report ErrEndOfStream until a Read to the
next Incoming message is made.
*/
package session
