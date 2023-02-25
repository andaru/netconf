package transport

import (
	"bufio"
	"io"

	"github.com/andaru/netconf/framing"
)

// Reader is a NETCONF transport decoder offering an io.Reader interface.
//
// NETCONF sessions exchange request/response and broadcast messages via
// the transport. Data sent on the wire is encoded according to the framing
// mode presently chosen; initially (and always, for :base:1.0 sessions),
// messages are sent verbatim and terminated by the end of message token.
// After the <hello> message, :base:1.1 sessions use a chunked framing
// mechanism with the same message semantics (see RFC6242, s4.2).
//
// The Reader decodes data using the current framing protocol, making
// it available to users via the Read call.
type Reader struct {
	src     *buckFilter
	eom     func()
	scanner *bufio.Scanner
	framing bufio.SplitFunc
	buf     []byte
}

// NewReader returns a new Reader given the source io.Reader and a function
// to be called after each message
func NewReader(source io.Reader, eomCallback func()) *Reader {
	if eomCallback == nil || source == nil {
		panic("NewReader: both source and eomCallback must be non-nil")
	}
	return &Reader{src: &buckFilter{src: source}, eom: eomCallback, framing: framing.SplitEOM(eomCallback)}
}

const (
	readerBufsize = 64 * 1024
)

// setup performs one time scanner setup
func (r *Reader) setup() {
	if r.scanner != nil {
		return
	}
	r.scanner = bufio.NewScanner(r.src)
	r.scanner.Buffer(make([]byte, readerBufsize), readerBufsize)
	r.scanner.Split(func(b []byte, eof bool) (int, []byte, error) { return r.framing(b, eof) })
}

func (r *Reader) Read(b []byte) (n int, err error) {
	r.setup()

	// evacuate the overflow buffer if present
	if len(r.buf) > 0 {
		n = copy(b, r.buf)
		r.buf = r.buf[n:]
		return
	}

	switch r.scanner.Scan() {
	case true:
		in := r.scanner.Bytes()
		if n = copy(b, in); n < len(in) {
			frm := in[n:]
			r.buf = make([]byte, len(frm))
			copy(r.buf, frm)
		}
		err = r.scanner.Err()
	default:
		if err = r.scanner.Err(); err == nil {
			err = io.EOF
		}
	}

	return
}

// SetFramingMode sets the NETCONF transport framing to end of message
// mode (chunked=false) or chunked framing mode (chunked=true).
func (r *Reader) SetFramingMode(chunked bool) {
	if r.src.disabled {
		panic("SetFramingMode must only be called once")
	}
	if chunked {
		r.framing = framing.SplitChunked(r.eom)
	} else {
		r.framing = framing.SplitEOM(r.eom)
	}
	// disable message bucker
	r.src.disabled = true
}
