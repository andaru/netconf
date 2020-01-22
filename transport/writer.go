package transport

import (
	"fmt"
	"io"
)

// Writer is a RFC6242 NETCONF transport encoder.
// An Encoder takes a destination io.Writer and encodes input
// appropriately, with a consistent API no matter the Encoder's
// current framing mode.
//
// It supports both RFC4742, NETCONF 1.0 end-of-message framing
// as well as NETCONF 1.1 chunked framing.
type Writer struct {
	dst     io.WriteCloser
	chunked bool
}

// NewWriter returns a new RFC6242 Encoder (inline filter)
// writing to the destination dst. Encoder implements io.Writer.
func NewWriter(dst io.WriteCloser) *Writer { return &Writer{dst: dst} }

// Write writes b to the Encoder's destination using the current framing mode.
func (w *Writer) Write(b []byte) (n int, err error) {
	if w.chunked {
		data := []byte(fmt.Sprintf("\n#%d\n%s", len(b), b))
		if n, err = w.dst.Write(data); err == nil && n < len(data) {
			err = io.ErrShortWrite
		}
		if err != nil {
			return 0, err
		}
		return len(b), nil
	}
	return w.dst.Write(b)
}

// WriteEnd writes the appropriate end of transmission message for the
// Encoder's current framing mode.  It must be called at the end of
// each request/response message sent by a NETCONF client or server.
//
// It returns the number of bytes written, along with any error.
func (w *Writer) WriteEnd() (int, error) {
	if w.chunked {
		return w.dst.Write([]byte("\n##\n"))
	}
	return w.dst.Write([]byte("]]>]]>"))
}

// Close closes the underlying writer
func (w *Writer) Close() error { return w.dst.Close() }

// SetFramingMode sets the framing mode of the Encoder. If chunked is
// true, chunked framing will be used, else end-of-message frming
// is used.
func (w *Writer) SetFramingMode(chunked bool) { w.chunked = chunked }
