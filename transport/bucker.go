package transport

import (
	"bytes"
	"io"
)

// buckFilter is a simple switchable input splitter/filter
//
// By default, it splits input around ']' characters
// to ease detection of :base:1.0 end-of-message markers.
// When buckNot is enabled, it flushes any previously
// split input before reading directly from the source.
type buckFilter struct {
	src     io.Reader
	bucked  []byte
	buckNot bool
}

func (bf *buckFilter) Read(p []byte) (n int, err error) {
	if len(bf.bucked) > 0 {
		n = copy(p, bf.bucked)
		bf.bucked = bf.bucked[n:]
		p = p[:n]
		return n, err
	}
	if n, err = bf.src.Read(p); bf.buckNot {
		return n, err
	}
	if idx := bytes.IndexByte(p, ']'); idx > -1 {
		bf.bucked = append(bf.bucked, p[idx:n]...)
		p = p[:idx]
		n = idx
	}
	return n, err
}

