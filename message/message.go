package message

import (
	"errors"
	"io"

	"github.com/andaru/nc2/transport"
)

// Decoder is a NETCONF message reader, implementing io.ReadCloser
type Decoder struct {
	D      *transport.Reader
	OnEOF  func()
	closed bool
	first  bool
}

var ErrEndOfStream = errors.New("end of stream")

func (d *Decoder) Read(p []byte) (n int, err error) {
	if d.closed {
		return 0, io.EOF
	}
	defer func() { d.first = false }()
	if n, err = d.D.Read(p); err == io.EOF {
		d.closed = true
		if d.first {
			return 0, ErrEndOfStream
		} else if d.OnEOF != nil {
			d.OnEOF()
			d.OnEOF = nil
		}
	}
	return
}

// Close closes the MessageReader
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

func (e *Encoder) Write(b []byte) (int, error) {
	if e.OnClosed == nil {
		return 0, io.ErrClosedPipe
	}
	e.written = true
	return e.E.Write(b)
}

// Close closes the MessageEncoder
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
