package schema

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/andaru/xmlstream"
)

// EventReceiver is the interface for handlers of NETCONF Schema events.
type EventReceiver interface {
	Init(context.Context) error

	RemoteCapabilities(context.Context, []string)
	RemoteSessionID(context.Context, string)
	HelloReceived(context.Context)

	RPC(context.Context, xml.StartElement)
	RPCError(context.Context, xml.StartElement)
	RPCReply(context.Context, xml.StartElement)
	Notification(context.Context, xml.StartElement)

	Finished(context.Context, error)
}

// EventSender is the interface for sending errors and other events to
// the (running) schema parser.
type EventSender interface {
	CriticalError(context.Context, error)
	Error(context.Context, error)
}

// EventSendReceiver is the combined event sender/receiver interface,
// used by parsers
type EventSendReceiver interface {
	EventSender
	EventReceiver
}

const (
	nsNC             = "urn:ietf:params:xml:ns:netconf:base:1.0"
	nsNCNotification = "urn:ietf:params:xml:ns:netconf:notification:1.0"
)

var (
	nameHello        = xml.Name{Space: nsNC, Local: "hello"}
	nameCapabilities = xml.Name{Space: nsNC, Local: "capabilities"}
	nameCapability   = xml.Name{Space: nsNC, Local: "capability"}
	nameSessionID    = xml.Name{Space: nsNC, Local: "session-id"}
	nameRPC          = xml.Name{Space: nsNC, Local: "rpc"}
	nameRPCReply     = xml.Name{Space: nsNC, Local: "rpc-reply"}
	nameRPCError     = xml.Name{Space: nsNC, Local: "rpc-error"}
	nameNotification = xml.Name{Space: nsNCNotification, Local: "notification"}

	nameWildcard = xml.Name{Space: "   ", Local: "        * "}
)

const (
	capBase10 = "urn:ietf:params:netconf:base:1.0"
	capBase11 = "urn:ietf:params:netconf:base:1.1"
)

type NETCONFSchema struct {
	bind func(ev EventSendReceiver) *xmlstream.Node
}

type Option func(*NETCONFSchema)

func NewClientSchema(opts ...Option) *NETCONFSchema {
	schema := &NETCONFSchema{bind: bindClientSession}
	for _, opt := range opts {
		opt(schema)
	}
	return schema
}

func NewServerSchema(opts ...Option) *NETCONFSchema {
	schema := &NETCONFSchema{bind: bindServerSession}
	for _, opt := range opts {
		opt(schema)
	}
	return schema
}

func (s *NETCONFSchema) Bind(ev EventSendReceiver) *xmlstream.Node {
	if s.bind == nil {
		panic("NETCONFSchema.bind must not be nil")
	}
	return s.bind(ev)
}

type ConstraintError struct {
	Node    *xmlstream.Node // Node is the schema node the constraint failed upon
	Name    string          // Name is the constraint which failed
	Message string
	Args    interface{}
	Token   xml.Token
}

func (e ConstraintError) Error() string {
	var constraint string
	switch e.Name {
	case "min-occurs", "max-occurs":
		v, _ := e.Args.([]int)
		constraint = fmt.Sprintf("%s:%d (saw %d)", e.Name, v[1], v[0])
	}
	if se, ok := e.Token.(xml.StartElement); ok && se.Name.Local != "" {
		return fmt.Sprintf("constraint %s failed on element %s", constraint, pprintElem(se))
	}
	return fmt.Sprintf("constraint %s failed", constraint)
}

func IsConstraintError(err error) (ConstraintError, bool) {
	ce, ok := err.(ConstraintError)
	return ce, ok
}

func validator(es EventSender) xmlstream.NodeTokenCallback {
	return func(ctx context.Context, n *xmlstream.Node, t xml.Token) {
		if want := n.Opt.MinOccurs(); want > -1 && want > n.Status.Occurs {
			es.Error(ctx, ConstraintError{
				Node:  n,
				Name:  "min-occurs",
				Token: xml.StartElement{Name: n.Name},
				Args:  []int{n.Status.Occurs, want}})
		}

		if want := n.Opt.MaxOccurs(); want > -1 && n.Status.Occurs > want {
			es.Error(ctx, ConstraintError{
				Node:  n,
				Name:  "max-occurs",
				Token: xml.StartElement{Name: n.Name},
				Args:  []int{n.Status.Occurs, want}})
		}
	}
}

func schemaNodeHello(ev EventSendReceiver, expectSessionID bool) *xmlstream.Node {
	validate := xmlstream.WithValidator(validator(ev))
	hello := xmlstream.ElementNode(nameHello, validate, xmlstream.WithMinOccurs(1), xmlstream.WithMaxOccurs(1))

	var remoteCapability []string
	hello.Append(
		xmlstream.EndElementEventNode(
			func(ctx context.Context, _ *xmlstream.Node, t xml.Token) {
				ev.RemoteCapabilities(ctx, remoteCapability)
				ev.HelloReceived(ctx)
			}))

	capabilities := hello.Append(xmlstream.ElementNode(nameCapabilities,
		xmlstream.WithMinOccurs(1), xmlstream.WithMaxOccurs(1), validate))
	capabilities.Validator = validator(ev)
	capability := capabilities.Append(xmlstream.ElementNode(nameCapability,
		xmlstream.WithMinOccurs(1), validate))
	capabilityCData := capability.Append(xmlstream.TextNode())
	capabilityCData.Append(
		xmlstream.TextEventNode(
			func(ctx context.Context, _ *xmlstream.Node, t xml.Token) {
				if cd, _ := t.(xml.CharData); len(cd) > 0 {
					remoteCapability = append(remoteCapability, string(cd))
				}
			}))

	if expectSessionID {
		sessionID := hello.Append(xmlstream.ElementNode(nameSessionID))
		sessionIDCData := sessionID.Append(xmlstream.TextNode())
		sessionIDCData.Append(
			xmlstream.TextEventNode(
				func(ctx context.Context, _ *xmlstream.Node, t xml.Token) {
					if cd, ok := t.(xml.CharData); ok {
						ev.RemoteSessionID(ctx, string(cd))
					}
				}))
	}

	return hello
}

func schemaNodeRPC(ev EventSendReceiver) *xmlstream.Node {
	rpc := xmlstream.ElementNode(nameRPC)
	rpc.Append(xmlstream.ElementNode(nameWildcard)).Append(
		xmlstream.StartElementEventNode(func(ctx context.Context, it *xmlstream.Node, t xml.Token) {
			ev.RPC(ctx, t.(xml.StartElement))
		}))
	return rpc
}

func schemaNodeRPCReply(ev EventSendReceiver) *xmlstream.Node {
	rpcReply := xmlstream.ElementNode(nameRPCReply)
	rpcReply.Append(xmlstream.ElementNode(nameWildcard)).Append(
		xmlstream.StartElementEventNode(func(ctx context.Context, it *xmlstream.Node, t xml.Token) {
			ev.RPCReply(ctx, t.(xml.StartElement))
		}))
	return rpcReply
}

func schemaNodeRPCError(ev EventSendReceiver) *xmlstream.Node {
	rpcError := xmlstream.ElementNode(nameRPCError)
	rpcError.Append(xmlstream.ElementNode(nameWildcard)).Append(
		xmlstream.StartElementEventNode(func(ctx context.Context, it *xmlstream.Node, t xml.Token) {
			ev.RPCError(ctx, t.(xml.StartElement))
		}))
	return rpcError
}

func schemaNodeNotification(ev EventSendReceiver) *xmlstream.Node {
	notif := xmlstream.ElementNode(nameNotification)
	notif.Append(xmlstream.ElementNode(nameWildcard)).Append(
		xmlstream.StartElementEventNode(func(ctx context.Context, it *xmlstream.Node, t xml.Token) {
			ev.RPCError(ctx, t.(xml.StartElement))
		}))
	return notif
}

func buildNETCONFSchema(ev EventSendReceiver, clientSession bool) *xmlstream.Node {
	nc := xmlstream.NewSchema()
	for _, node := range []*xmlstream.Node{
		schemaNodeHello(ev, clientSession),
		schemaNodeRPC(ev),
	} {
		nc.Append(node)
	}
	if clientSession {
		nc.Append(schemaNodeRPCError(ev))
		nc.Append(schemaNodeRPCReply(ev))
		nc.Append(schemaNodeNotification(ev))
	}
	return nc
}

// Parser is a NETCONF stream parser conforming to a schema.
//
// Schema must be a non-nil client or server schema object.
type Parser struct {
	Schema  *NETCONFSchema
	EV      EventSendReceiver
	Cancel  context.CancelFunc
	SM      *xmlstream.StateMachine
	running chan error
}

// Run the parser using the provided context and the XML decoder.
// Events will be sent concurrently to the events channel, which must
// not be nil.
//
// If there is no initialization error, will return nil after starting
// the parser in a goroutine. Once running, call Wait() to determine
// when the parser has completed processing input. Schema validation
// will be performed by the parser after processing completes.
func (p *Parser) Run(ctx context.Context, d *xml.Decoder) error {
	p.running = make(chan error, 1)
	if err := p.EV.Init(ctx); err != nil {
		// critical; send an error to then close p.running to unblock Wait() callers
		p.running <- err
		close(p.running)
		return err
	}

	schema := p.Schema.Bind(p.EV)
	p.SM = xmlstream.New(ParseSchemaXML(schema, d, p.EV))
	p.SM.End = func(ctx context.Context, sm *xmlstream.StateMachine) xmlstream.StateFn {
		defer close(p.running)
		if p.SM.Error == io.EOF {
			p.SM.Error = nil
		}
		p.EV.Finished(ctx, p.SM.Error)
		// perform schema validation prior to closing the event channel
		ValidateSchema(ctx, schema, p.EV)
		return nil
	}

	go p.SM.Run(ctx)
	return p.SM.Error
}

func (p *Parser) Wait() error { return <-p.running }

var (
	bindServerSession = func(ev EventSendReceiver) *xmlstream.Node {
		return buildNETCONFSchema(ev, false)
	}
	bindClientSession = func(ev EventSendReceiver) *xmlstream.Node {
		return buildNETCONFSchema(ev, true)
	}
)
