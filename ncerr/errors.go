package ncerr

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
)

// Type represents the NETCONF error-type enumerate
type Type int

const (
	// TypeApplication is an application layer error
	TypeApplication Type = iota
	// TypeProtocol is a NETCONF protocol layer error
	TypeProtocol
	// TypeRPC is a NETCONF RPC layer error
	TypeRPC
	// TypeTransport is an error at the secure transport layer
	TypeTransport
)

func (t Type) String() string {
	switch t {
	case TypeApplication:
		return "application"
	case TypeProtocol:
		return "protocol"
	case TypeRPC:
		return "rpc"
	case TypeTransport:
		return "transport"
	default:
		return fmt.Sprintf("Type(%d)", int(t))
	}
}

func (t *Type) UnmarshalText(b []byte) error {
	b = bytes.TrimSpace(b)
	switch string(b) {
	case "application":
		*t = TypeApplication
	case "protocol":
		*t = TypeProtocol
	case "rpc":
		*t = TypeRPC
	case "transport":
		*t = TypeTransport
	default:
		return errors.New("unknown value")
	}
	return nil
}

func (t Type) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

// Severity represents the NETCONF error-severity enumerate
type Severity int

const (
	// SeverityError indicates "error" level
	SeverityError Severity = iota
	// SeverityWarning indicates "warning" level.
	// (Not used in errors defined in RFC6241 Appendix A)
	SeverityWarning
)

func (s Severity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return fmt.Sprintf("Severity(%d)", int(s))
	}
}

func (s Severity) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Severity) UnmarshalText(b []byte) error {
	b = bytes.TrimSpace(b)
	switch string(b) {
	case "error":
		*s = SeverityError
	case "warning":
		*s = SeverityWarning
	default:
		return errors.New("unknown value")
	}
	return nil
}

// Error represents a NETCONF error.
//
// XMLName must be set prior to calls to xml.Marshal.
// For example, to frame the error as an <rpc-error>:
//   e := &Error{}
//   e.XMLName = xml.Name{Space: "urn:ietf:params:xml:ns:netconf:base:1.0", Local:"rpc-error"}
//   out, _ := xml.Marshal(e)
type Error struct {
	XMLName  xml.Name          `xml:"urn:ietf:params:xml:ns:netconf:base:1.0 rpc-error" json:"-"`
	Type     Type              `xml:"error-type" json:"error-type"`
	Tag      string            `xml:"error-tag" json:"error-tag"`
	Severity Severity          `xml:"error-severity" json:"error-severity"`
	AppTag   string            `xml:"error-app-tag,omitempty" json:"error-app-tag,omitempty"`
	Path     string            `xml:"error-path,omitempty" json:"error-path,omitempty"`
	Message  string            `xml:"error-message,omitempty" json:"error-message,omitempty"`
	Info     *rfc6241errorInfo `xml:"error-info,omitempty" json:"error-info,omitempty"`
}

type rfc6241errorInfo struct {
	BadAttribute string `xml:"bad-attribute,omitempty" json:"bad-attribute,omitempty"`
	BadElement   string `xml:"bad-element,omitempty" json:"bad-element,omitempty"`
	BadNamespace string `xml:"bad-namespace,omitempty" json:"bad-namespace,omitempty"`
	SessionID    string `xml:"session-id,omitempty" json:"session-id,omitempty"`
}

func (e Error) Error() string {
	s := fmt.Sprintf("%s error tag:%s", e.Type, e.Tag)
	if e.AppTag != "" {
		s += "app-tag:" + e.AppTag
	}
	if e.Path != "" {
		s += " path:" + e.Path
	}
	if info := e.Info; info != nil {
		if info.BadAttribute != "" {
			s += " bad-attribute:" + info.BadAttribute
		}
		if info.BadElement != "" {
			s += " bad-element:" + info.BadElement
		}
		if info.BadNamespace != "" {
			s += " bad-namespace:" + info.BadNamespace
		}
		if info.SessionID != "" {
			s += " session-id:" + info.SessionID
		}
	}
	if e.Message != "" {
		s += " " + e.Message
	}
	return s
}

func InUse(opts ...Option) *Error {
	e := &Error{Tag: "in-use"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func InvalidValue(opts ...Option) *Error {
	e := &Error{Tag: "invalid-value"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func TooBig(opts ...Option) *Error {
	e := &Error{Tag: "too-big"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func MissingAttribute(attributeName, elementName string, opts ...Option) *Error {
	e := &Error{
		Tag:  "missing-attribute",
		Info: &rfc6241errorInfo{BadAttribute: attributeName, BadElement: elementName},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func BadAttribute(attributeName, elementName string, opts ...Option) *Error {
	e := &Error{
		Tag:  "bad-attribute",
		Info: &rfc6241errorInfo{BadAttribute: attributeName, BadElement: elementName},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func UnknownAttribute(attributeName, elementName string, opts ...Option) *Error {
	e := &Error{
		Tag:  "unknown-attribute",
		Info: &rfc6241errorInfo{BadAttribute: attributeName, BadElement: elementName},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func MissingElement(elementName string, opts ...Option) *Error {
	e := &Error{
		Tag:  "missing-element",
		Info: &rfc6241errorInfo{BadElement: elementName},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func BadElement(elementName string, opts ...Option) *Error {
	e := &Error{
		Tag:  "bad-element",
		Info: &rfc6241errorInfo{BadElement: elementName},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func UnknownElement(elementName string, opts ...Option) *Error {
	e := &Error{
		Tag:  "unknown-element",
		Info: &rfc6241errorInfo{BadElement: elementName},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func UnknownNamespace(elementName, namespace string, opts ...Option) *Error {
	e := &Error{
		Tag:  "unknown-namespace",
		Info: &rfc6241errorInfo{BadElement: elementName, BadNamespace: namespace},
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func AccessDenied(opts ...Option) *Error {
	e := &Error{Tag: "access-denied"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func LockDenied(sessionID string, opts ...Option) *Error {
	e := &Error{
		Tag:  "lock-denied",
		Info: &rfc6241errorInfo{SessionID: sessionID},
	}
	for _, opt := range opts {
		opt(e)
	}
	// error-type must be protocol for lock-denied
	e.Type = TypeProtocol

	return e
}

func ResourceDenied(opts ...Option) *Error {
	e := &Error{Tag: "resource-denied"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func RollbackFailed(opts ...Option) *Error {
	e := &Error{Tag: "rollback-failed"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func DataExists(opts ...Option) *Error {
	e := &Error{Tag: "data-exists"}
	for _, opt := range opts {
		opt(e)
	}
	// error-type must be application for data-exists
	e.Type = TypeApplication
	return e
}

func DataMissing(opts ...Option) *Error {
	e := &Error{Tag: "data-missing"}
	for _, opt := range opts {
		opt(e)
	}
	// error-type must be application for data-missing
	e.Type = TypeApplication
	return e
}

func OperationNotSupported(opts ...Option) *Error {
	e := &Error{Tag: "operation-not-supported"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func OperationFailed(opts ...Option) *Error {
	e := &Error{Tag: "operation-failed"}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

func MalformedMessage(opts ...Option) *Error {
	e := &Error{Tag: "malformed-message"}
	for _, opt := range opts {
		opt(e)
	}
	// error-type must be rpc for malformed-message
	e.Type = TypeRPC
	return e
}
