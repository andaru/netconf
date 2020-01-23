package message

import (
	"github.com/andaru/netconf/transport"
)

// Splitter manages the NETCONF transport session for the message layer.
//
// Its task is to split the session up into a series of NETCONF messages.
// A NETCONF message is a complete XML document. Normally, a single NETCONF
// message is followed by the transport end-of-message marker. If the
// session has negotiated pipelining support, multiple NETCONF messages
// (typically multiple <rpc> or <rpc-reply> elemenets) may be exchanged
// before the transport end-of-message marker is exchanged.
//
// The Splitter provides access to the current message Decoder and Encoder.
// The Reader will return EOF when the end-of-message marker is received,
// while the Writer sends the end-of-message marker when closed.
type Splitter struct {
	R *transport.Reader
	W *transport.Writer

	dec    *Decoder
	enc    *Encoder
	newDec bool
	newEnc bool
}

// Reader returns the current message's reader (implementing io.Reader),
// or creates a new message reader and returns that.
func (s *Splitter) Reader() *Decoder {
	if s.dec == nil || s.newDec {
		s.dec = &Decoder{D: s.R, OnEOF: s.FinishReader, first: true}
		s.newDec = false
	}
	return s.dec
}

// Writer returns the current message's writer (implementing io.WriteCloser)
func (s *Splitter) Writer() *Encoder {
	if s.enc == nil || s.newEnc {
		s.enc = &Encoder{E: s.W, OnClosed: s.FinishWriter}
		s.newEnc = false
	}
	return s.enc
}

// FinishWriter emits a new writer on the next call(s) to Writer
func (s *Splitter) FinishWriter() { s.newEnc = true }

// FinishReader emits a new reader on the next call(s) to Reader
func (s *Splitter) FinishReader() { s.newDec = true }
