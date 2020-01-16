package transport

import (
	"bufio"
	"fmt"
	"io"

	"bytes"

	"github.com/andaru/netconf/framing"
)

type buckFilter struct {
	src        io.Reader
	buckNoMore bool
	bucked     []byte
}

func (bf *buckFilter) Read(p []byte) (n int, err error) {
	if len(bf.bucked) > 0 {
		n = copy(p, bf.bucked)
		bf.bucked = bf.bucked[n:]
		p = p[:n]
		return n, err
	}
	if n, err = bf.src.Read(p); bf.buckNoMore {
		return n, err
	}
	if idx := bytes.IndexByte(p, ']'); idx > -1 {
		bf.bucked = append(bf.bucked, p[idx:n]...)
		p = p[:idx]
		n = idx
	}
	return n, err
}

// Decoder is an RFC6242 NETCONF transport decoder.
// A Decoder takes a source io.Reader and offers an io.Reader
// interface to read decoded transport stream data.
//
// It supports both "end of message" (NETCONF 1.0 backwards
// compatible) and "chunked framing" encoding (NETCONF 1.1)
// formats.
type Decoder struct {
	src     *buckFilter
	f       bufio.SplitFunc
	scanner *bufio.Scanner
	cur     []byte
	bufsize int
}

// DecoderOption is a constructor option for a Decoder
type DecoderOption func(*Decoder)

// WithDecoderBufferSize sets the maximum buffer size to use for the Decoder.
//
// Size has a floor of 16 bytes; values lower than that will return a 16 byte
// buffer size option.
func WithDecoderBufferSize(size int) DecoderOption {
	return func(d *Decoder) {
		if size < 6 {
			size = 6
		}
		d.bufsize = size
	}
}

// WithEOMCallback sets callback to be called by the Decoder when any end-of-message
// framing token is seen. Applications typically use this to determine whether to call
// SetChunkedFraming on the Decoder
func WithEOMCallback(callback func()) DecoderOption {
	return func(d *Decoder) { d.f = framing.SplitEOM(callback) }
}

// NewDecoder returns a new Decoder reading from src, configured by opts
func NewDecoder(src io.Reader, opts ...DecoderOption) *Decoder {
	d := &Decoder{src: &buckFilter{src: src}, bufsize: 4096, f: framing.SplitEOM(nil)}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// Read reads from the Decoder into b. It returns the number of bytes read into
// b from the transport stream as well as any error. n may be non-zero and err
// may be non-nil for the same call, indicating b contains some data read prior
// to the error (potentially common for error io.EOF).  Implements io.Reader.
func (d *Decoder) Read(b []byte) (n int, err error) {
	d.setup()

	// evacuate the overflow buffer if present
	if len(d.cur) > 0 {
		n = copy(b, d.cur)
		d.cur = d.cur[n:]
		return
	}

	switch d.scanner.Scan() {
	case true:
		in := d.scanner.Bytes()
		if n = copy(b, in); n < len(in) {
			frm := in[n:]
			d.cur = make([]byte, len(frm))
			copy(d.cur, frm)
		}
		err = d.scanner.Err()
	default:
		if err = d.scanner.Err(); err == nil {
			err = io.EOF
		}
	}

	return
}

func (d *Decoder) split(b []byte, atEOF bool) (int, []byte, error) {
	n, t, err := d.f(b, atEOF)
	return n, t, err
}

func (d *Decoder) setup() {
	if d.scanner == nil && d.src != nil {
		d.scanner = bufio.NewScanner(d.src)
		d.scanner.Buffer(make([]byte, d.bufsize), d.bufsize)
		d.scanner.Split(d.split)
	}
}

// SetFramingMode sets the Decoder transport's framing mode.
// If passed true, the decoder will switch to chunked mode immediately.
//
// This function must be called immediately after reading the </hello> element
// from the Decoder if and only if the ":base:1.1" <capability> was present within.
func (d *Decoder) SetFramingMode(chunked bool) {
	if chunked {
		d.f = framing.SplitChunked()
	}
	d.src.buckNoMore = true
}

// Encoder is an RFC6242 NETCONF transport encoder.
// An Encoder takes a destination io.Writer and encodes input
// appropriately, with a consistent API no matter the Encoder's
// current framing mode.
//
// It supports both RFC4742, NETCONF 1.0 end-of-message framing
// as well as NETCONF 1.1 chunked framing.
type Encoder struct {
	dst     io.Writer
	chunked bool
}

// NewEncoder returns a new RFC6242 Encoder (inline filter)
// writing to the destination dst. Encoder implements io.Writer.
func NewEncoder(dst io.Writer) *Encoder { return &Encoder{dst: dst} }

// Write writes b to the Encoder's destination using the current framing mode.
func (e *Encoder) Write(b []byte) (int, error) {
	if e.chunked {
		return e.dst.Write([]byte(fmt.Sprintf("\n#%d\n%s", len(b), b)))
	}
	return e.dst.Write(b)
}

// WriteEnd writes the appropriate end of transmission message for the
// Encoder's current framing mode.  It must be called at the end of
// each request/response message sent by a NETCONF client or server.
//
// It returns the number of bytes written, along with any error.
func (e *Encoder) WriteEnd() (int, error) {
	if e.chunked {
		return e.dst.Write([]byte("\n##\n"))
	}
	return e.dst.Write([]byte("]]>]]>"))
}

// SetChunkedFraming enables chunked framing on the Encoder.
//
// Called after receiving </hello> when the :base:1.1 capability
// was provided in the <hello> element.
func (e *Encoder) SetChunkedFraming() { e.chunked = true }
