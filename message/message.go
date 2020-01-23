package message

import (
	"errors"
	"io"

	"github.com/andaru/netconf/transport"
)

// Decoder is a NETCONF message reader, implementing io.ReadCloser.
//
// D must point to an initialized *transport.Reader, while OnEOF must
// be a non-nil function, called at end of stream, when the underlying
// transport sees EOF.
type Decoder struct {
	D      *transport.Reader
	OnEOF  func()
	closed bool
	first  bool
}

// ErrEndOfStream indicates the transport stream has ended.
// This is somewhat equivalent to io.EOF, in that it indicates
// no more data can be read from the underlying transport due
// to an EOF on said transport.
var ErrEndOfStream = errors.New("end of stream")

// Read reads data from the underlying transport into, decoding
// the transport framing and populating p with uncooked data,
// implementing io.Reader.
func (d *Decoder) Read(p []byte) (n int, err error) {
	if d.closed {
		return 0, io.EOF
	}
	defer func() { d.first = false }()
	if n, err = d.D.Read(p); err == io.EOF {
		d.closed = true
		if d.first {
			err = ErrEndOfStream
		} else if d.OnEOF != nil {
			d.OnEOF()
			d.OnEOF = nil
		}
	}
	return
}

// Close closes the message. Further reads to this channel will return EOF.
func (d *Decoder) Close() error {
	d.closed = true
	return nil
}

// Encoder is a NETCONF message encoder, implementing io.WriteCloser
type Encoder struct {
	E        *transport.Writer
	OnClosed func()
	written  bool
}

// Write writes the cooked encoding of b in the current framing
// mode to the underlying transport, implementing io.Writer.
func (e *Encoder) Write(b []byte) (int, error) {
	if e.OnClosed == nil {
		return 0, io.ErrClosedPipe
	}
	e.written = true
	return e.E.Write(b)
}

// Close closes the message, causing the end of message token
// to be written to the underlying transport if any data had
// been written to this channel.
func (e *Encoder) Close() (err error) {
	if e.OnClosed != nil {
		if e.written {
			_, err = e.E.WriteEnd()
		}
		e.OnClosed()
		e.OnClosed = nil
	}
	return err
}
