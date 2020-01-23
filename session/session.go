package session

import (
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/andaru/netconf/message"
	"github.com/andaru/netconf/transport"
	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
)

// New returns a new NETCONF Session
func New(src io.Reader, dst io.WriteCloser, config Config) *Session {
	s := &Session{
		Config: &config,
		State:  &State{},
		src:    src,
		dst:    dst,
	}
	s.reader = transport.NewReader(src, s.onEndOfMessage)
	s.writer = transport.NewWriter(dst)
	s.Message = &message.Splitter{R: s.reader, W: s.writer}
	return s
}

// Run executes the Session s, using Handler h
func Run(s *Session, h Handler) {
	// perform the <hello> and <capabilities> exchange
	if s.InitialHandshake() {
		// session was established, run the established callback
		h.OnEstablish(s)
		// call the session message callback while the session remains established
		for s.State.Status == StatusEstablished {
			h.OnMessage(s)
		}
	}
	if s.State.Status == StatusError {
		// session failed to establish, run the error callback
		h.OnError(s)
	}
	// close the session
	s.Close()
	// finally, run the close callback
	h.OnClose(s)
}

// Session represents a NETCONF session
type Session struct {
	Config *Config
	State  *State

	src     io.Reader
	dst     io.WriteCloser
	reader  *transport.Reader
	writer  *transport.Writer
	Message *message.Splitter
}

// Handler is the Session handler interface.
// Client and/or server applications implement this interface.
//
// See Run() for usage.
type Handler interface {
	// OnEstablish is called when the session is established.
	// When called, session capabilities processing has completed
	OnEstablish(*Session)
	// OnMessage is called after the session is established
	// and then repeatedly while the session remains in the
	// StatusEstablished state.
	OnMessage(*Session)
	// OnError is called once if the session transitions from
	// to the StatusError state.  This can occur initially,
	// instead of OnEstablish, or after the OnEstablish state,
	// indicating a session transport error occurred.
	OnError(*Session)
	// OnClose is called immediately after the session's
	// transport is closed.
	OnClose(*Session)
}

// Config contains Session configuration
type Config struct {
	// ID is the configured session-id. Must be 0 for client sessions
	// and non-0 for server sessions
	ID uint32
	// Capabilities holds our session capabilities
	Capabilities Capabilities
}

// HandlerFunc is a Session handler function
type HandlerFunc func(*Session)

// State contains runtime Session state
type State struct {
	// ID is the established session-id. Will be populated during
	// capabilities exchange.
	ID uint32
	// Capabilities holds the remote peer's capabilities
	Capabilities Capabilities
	// Status is the session status
	Status Status
	// Counterts contains session counters
	Counters struct {
		// RxMsgs is the number of NETCONF messages received on the session
		RxMsgs int
	}

	// Opaque is user private data and is not used by the netconf libraries.
	Opaque interface{}

	errs []error
}

// Status is a Session's (present) state.
type Status int

const (
	// StatusInactive is the initial session state, indicating that
	// I/O has not yet been started.
	StatusInactive Status = iota
	// StatusCapabilitiesExchange is the capabilities exchange status.
	// This is set after the <hello> message is received from the peer.
	StatusCapabilitiesExchange
	// StatusEstablished is set after capabilities exchange finishes
	// if the session has been successfully established. Otherwise,
	// the state machine will proceed to StatusError.
	StatusEstablished

	// StatusError indicates the session has encountered an error.
	StatusError
	// StatusClosed indicates the session closed normally.
	StatusClosed
)

// Incoming returns the incoming (from peer) message channel (implements io.Reader).
//
// This function always returned a non-nil message channel.
//
// Note that the object returned returned will return message.ErrEndOfStream to Read when
// the underlying transport reaches EOF, and will return io.EOF at the end of each message.
func (s *Session) Incoming() *message.Decoder { return s.Message.Reader() }

// Outgoing returns the outgoing (to peer) message channel (implements io.WriteCloser).
//
// This function always returned a non-nil message channel.
func (s *Session) Outgoing() *message.Encoder { return s.Message.Writer() }

// InitialHandshake performs session handshake, capabilities exchange and framing mode selection.
//
// Returns true if the handshake completed successfully, in which
// case the session status will be StatusEstablished; otherwise returns false
// if an error occurred (for either transport or validation reasons), in
// which case Session.Errors will return non-nil and the session status will
// be StatusError.
func (s *Session) InitialHandshake() (ok bool) {
	if s.State.Status == StatusInactive {
		s.State.Status = StatusCapabilitiesExchange
		if s.sendHello(); len(s.State.errs) == 0 {
			s.recvHello()
		}
		ok = len(s.State.errs) == 0
	}
	if !ok {
		s.State.Status = StatusError
	}
	return ok
}

// Close closes the Session
func (s *Session) Close() error {
	s.State.Status = StatusClosed
	s.Outgoing().Close()
	s.Incoming().Close()
	err := s.dst.Close()
	if err == io.ErrClosedPipe {
		err = nil
	}
	return err
}

// AddError adds an error to the session state
func (s *Session) AddError(errs ...error) (added int) {
	for _, err := range errs {
		if err != nil {
			s.State.errs = append(s.State.errs, err)
			added++
		}
	}
	return added
}

// Errors returns all session errors
func (s *Session) Errors() []error { return s.State.errs }

// Run executes the session using Handler h
func (s *Session) Run(handler Handler) { Run(s, handler) }

// onEndOfMessage performs end-of-message handling
func (s *Session) onEndOfMessage() {
	s.State.Counters.RxMsgs++
	// close the incoming message and rotate it at the next read
	s.Incoming().Close()
	s.Message.FinishReader()
}

// doCapabilitiesExchange performs capability exchange handling,
// including setting the session framing mode based on the :base:1.x
// capability seen
func (s *Session) doCapabilitiesExchange() {
	sc, pc := s.Config.Capabilities, s.State.Capabilities
	base11 := sc.Has(capBase11) && pc.Has(capBase11)
	if base10 := sc.Has(capBase10) && pc.Has(capBase10); !(base11 || base10) {
		s.AddError(errors.New("session failed to negotiate framing mode"))
		s.State.Status = StatusError
		return
	}
	s.reader.SetFramingMode(base11)
	s.writer.SetFramingMode(base11)
	s.State.Status = StatusEstablished
}

func (s *Session) recvHello() {
	// parse the incoming message's XML document
	doc, parseErr := xmlquery.Parse(s.Incoming())
	if s.AddError(parseErr) > 0 {
		s.State.Status = StatusError
		return
	}
	// look for a <hello> element in the NETCONF namespace
	hello := xmlquery.QuerySelector(doc, xpNSetHello)
	if hello == nil {
		s.AddError(errors.New("missing <hello> element"))
		s.State.Status = StatusError
		return
	}

	// add capabilities
	results := xmlquery.QuerySelectorAll(doc, xpNSetCapability)
	for _, capability := range results {
		if x := strings.TrimSpace(capability.InnerText()); x != "" {
			s.State.Capabilities = append(s.State.Capabilities, x)
		}
	}
	// error if no capabilities were found (we must have seen at least
	// :base:1.0 or :base:1.1 per the specification)
	if len(s.State.Capabilities) == 0 {
		s.AddError(errors.New("missing non-empty <capability> element(s)"))
		s.State.Status = StatusError
		return
	}

	// validate the existence and value of any session-id element.
	// The spec states only a client should receive a <session-id>
	// element in the <hello>.  If we have a non-zero s.Config.ID,
	// we are a server session and should not receive one.
	var err error
	switch sid := xmlquery.QuerySelector(doc, xpNSetSessionID); {
	case sid == nil && s.Config.ID == 0:
		// no session-id received but we consider ourselves a client
		err = errors.New("no session-id received for client session")
	case sid != nil && s.Config.ID != 0:
		// we are a server session but the peer sent us a session-id
		err = errors.New("session-id received from client peer")
	case sid != nil:
		// client session: session status takes the received session ID
		if idVal := strings.TrimSpace(sid.InnerText()); idVal != "" {
			v, perr := strconv.ParseUint(idVal, 10, 32)
			if perr != nil {
				err = errors.New("invalid session-id value")
			}
			s.State.ID = uint32(v)
		} else {
			err = errors.New("missing session-id value")
		}
	case sid == nil:
		// server session: the session status takes our configured ID
		s.State.ID = s.Config.ID
	}
	if s.AddError(err) > 0 {
		s.State.Status = StatusError
	}

	// perform capabilities exchange (inc. :base:1.x protocol selection)
	if err == nil {
		s.doCapabilitiesExchange()
	}
}

// sendHello sends the <hello> message along with any capabilities
// configured on the session and the session-id element if the
// configured ID is non-zero.
func (s *Session) sendHello() {
	defer func() { s.AddError(s.Outgoing().Close()) }()
	xe := xml.NewEncoder(s.Outgoing())
	err := xe.EncodeToken(seHello)
	if err == nil {
		err = xe.EncodeToken(seCapabilities)
	}
	for _, cap := range s.Config.Capabilities {
		if err != nil {
			break
		}
		err = xe.EncodeToken(seCapability)
		err = xe.EncodeToken(xml.CharData(cap))
		err = xe.EncodeToken(seCapability.End())
	}
	if err == nil && s.Config.ID != 0 {
		err = xe.EncodeToken(seSessionID)
		err = xe.EncodeToken(xml.CharData(strconv.FormatUint(uint64(s.Config.ID), 10)))
		err = xe.EncodeToken(seSessionID.End())
	}
	if err == nil {
		err = xe.EncodeToken(seCapabilities.End())
		err = xe.EncodeToken(seHello.End())
		err = xe.Flush()
	}
	if s.AddError(err) > 0 {
		s.State.Status = StatusError
	}
}

func xn(local string, spaces ...string) xml.Name {
	n := xml.Name{Local: local}
	for _, space := range spaces {
		n.Space = space
		return n
	}
	return n
}

const (
	capBase10 = "urn:ietf:params:netconf:base:1.0"
	capBase11 = "urn:ietf:params:netconf:base:1.1"

	xmlnsNetconf = "urn:ietf:params:xml:ns:netconf:base:1.0"
)

var (
	xpNSetHello      = xpath.MustCompile(`/hello[namespace-uri()='urn:ietf:params:xml:ns:netconf:base:1.0']`)
	xpNSetCapability = xpath.MustCompile(`/hello[namespace-uri()='urn:ietf:params:xml:ns:netconf:base:1.0']/capabilities/capability`)
	xpNSetSessionID  = xpath.MustCompile(`/hello[namespace-uri()='urn:ietf:params:xml:ns:netconf:base:1.0']/session-id`)

	seHello        = xml.StartElement{Name: xn("hello", xmlnsNetconf)}
	seCapabilities = xml.StartElement{Name: xn("capabilities")}
	seCapability   = xml.StartElement{Name: xn("capability")}
	seSessionID    = xml.StartElement{Name: xn("session-id")}
)
