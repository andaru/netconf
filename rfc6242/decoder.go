package rfc6242

import (
	"bufio"
	"io"
	"sync"

	"github.com/pkg/errors"
)

// FramerFn is the input tokenization function used by a Decoder.
type FramerFn func(d *Decoder, data []byte, atEOF bool) (advance int, token []byte, err error)

// Decoder is an RFC6242 transport framing decoder filter.
//
// Decoder operates as an inline filter, taking a io.Reader as input
// and providing io.Reader as well as the low-overhead io.WriterTo.
//
// When used with io.Copy, the io.Reader interface is expected of the
// source argument, but if the value passed also supports io.WriterTo
// (as does Decoder), the WriteTo function will instead be called with
// the copy destination as its writer argument.
type Decoder struct {
	// Input is the input source for the Decoder. The input stream
	// must consist of RFC6242 encoded data according to the current
	// Framer.
	Input io.Reader
	// Framer is the function used to tokenize incoming input. It may
	// be changed prior to any call to Read, to enable the :base:1.0
	// to :base:1.1 NETCONF protocol upgrade described in
	// RFC6241/RFC6242.
	Framer FramerFn

	s    *bufio.Scanner
	pr   *io.PipeReader
	pw   *io.PipeWriter
	once *sync.Once

	// decoder state used in chunked framing mode
	scanErr       error
	chunkDataLeft uint64
	bufSize       int
	anySeen       bool
	eofOK         bool
}

// NewDecoder creates a new RFC6242 transport framing decoder
func NewDecoder(input io.Reader, options ...DecoderOption) *Decoder {
	d := &Decoder{
		Input:   input,
		Framer:  DecoderEndOfMessage,
		bufSize: defaultReaderBufferSize,
		s:       bufio.NewScanner(input),
		once:    new(sync.Once),
	}
	for _, option := range options {
		option(d)
	}
	d.pr, d.pw = io.Pipe()
	tmp := make([]byte, d.bufSize)
	d.s.Buffer(tmp, d.bufSize)
	d.s.Split(d.split)
	return d
}

// Read reads from the Decoder's input and copies the data into b,
// implementing io.Reader.
func (d *Decoder) Read(b []byte) (int, error) {
	d.once.Do(func() { go readerLoop(d) })
	n, err := d.pr.Read(b)
	if err == io.EOF && !d.eofOK {
		err = errors.WithStack(io.ErrUnexpectedEOF)
	}
	return n, err
}

// WriteTo reads from the Decoder's input, strips the transport
// encoding and writes the decoded data to w, implementing
// io.WriterTo.
func (d *Decoder) WriteTo(w io.Writer) (n int64, err error) {
	var wn int
	for err == nil && d.s.Scan() {
		b := d.s.Bytes()
		wn, err = w.Write(b)
		n += int64(wn)
	}
	err = d.s.Err()
	if err == nil && !d.eofOK {
		err = errors.WithStack(io.ErrUnexpectedEOF)
	}
	return
}

func (d *Decoder) split(b []byte, eof bool) (a int, t []byte, err error) {
	if eof && len(b) == 0 {
		err = d.scanErr
		return
	}
	return d.Framer(d, b, eof)
}

func readerLoop(d *Decoder) (n int64, err error) {
	var wn int
	for err == nil && d.s.Scan() {
		if scan := d.s.Bytes(); len(scan) > 0 {
			wn, err = d.pw.Write(scan)
			n += int64(wn)
		}
	}
	d.pw.CloseWithError(d.s.Err())
	return
}

const (
	// RFC6242 section 4.2 defines the "maximum allowed chunk-size".
	rfc6242maximumAllowedChunkSize = 4294967295
	// the length of `rfc6242maximumAllowedChunkSize` in bytes on the wire.
	rfc6242maximumAllowedChunkSizeLength = 10
	// defaultReaderBufferSize is the default read buffer capacity size.
	defaultReaderBufferSize = 65536
)
