package framing

import (
	"bytes"
	"io"
	"strconv"

	"bufio"

	"github.com/pkg/errors"
)

// SplitEOM returns a bufio.SplitFunc suitable for RFC6242
// "end-of-message delimited" NETCONF transport streams.
func SplitEOM(cb func()) bufio.SplitFunc {
	var eofOK, seen bool
	return func(b []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(b) == 0 {
			if !eofOK && seen {
				err = io.ErrUnexpectedEOF
			}
			return
		}
		seen = true
		idx := bytes.Index(b, tokenEOM)
		if idx > -1 {
			if cb != nil {
				cb()
			}
			advance = idx + len(tokenEOM)
			token = b[:idx]
			eofOK = true
			return
		}
		eofOK = false
		if len(b) <= (len(tokenEOM)*2) && !atEOF {
			return 0, nil, nil
		}
		idx = bytes.IndexByte(b, ']')
		if idx > -1 {
			advance = idx
			token = b[:idx]
		} else {
			advance = len(b)
			token = b
		}
		return
	}
}

// SplitChunked returns a bufio.SplitFunc suitable for decoding
// "chunked framing" NETCONF transport streams.
//
// It must only be used with bufio.Scanner who have a buffer of
// at least 16 bytes (rarely is this a concern).
func SplitChunked(cb func()) bufio.SplitFunc {
	type stateT int
	const (
		headerStart stateT = iota
		headerSize
		data
		endOfChunks
	)
	var state stateT
	var cs, dataleft int

	return func(b []byte, atEOF bool) (advance int, token []byte, err error) {
		for cur := b[advance:]; err == nil && advance < len(b); cur = b[advance:] {
			if len(cur) < 4 && !atEOF {
				return
			}
			switch state {
			case headerStart:
				switch {
				case bytes.HasPrefix(cur, []byte("\n#")):
					if len(cur) < 4 {
						err = ErrBadChunk
						return
					}
					switch r := cur[2]; {
					case r == '#':
						advance += 3
						state = endOfChunks
					case r >= '1' && r <= '9':
						advance += 2
						state = headerSize
					default:
						err = ErrBadChunk
					}
				default:
					err = ErrBadChunk
				}
			case headerSize:
				switch idx := bytes.IndexByte(cur, '\n'); {
				case idx < 1, idx > 10:
					if len(cur) < 11 && !atEOF {
						return
					}
					err = ErrBadChunk
				default:
					csize := cur[:idx]
					if csizeVal, csizeErr := strconv.ParseUint(string(csize), 10, 31); csizeErr != nil {
						err = ErrBadChunk
					} else {
						advance += idx + 1
						dataleft = int(csizeVal)
						state = data
					}
				}
			case data:
				var rsize int
				if rsize = len(cur); dataleft < rsize {
					rsize = dataleft
				}
				token = append(token, cur[:rsize]...)
				advance += rsize
				if dataleft -= rsize; dataleft < 1 {
					state = headerStart
					cs++
				}
				if rsize > 0 {
					return
				}
			case endOfChunks:
				switch r := cur[0]; {
				case r == '\n' && cs > 0:
					advance++
					state = headerStart
					if cb != nil {
						cb()
					}
				default:
					err = ErrBadChunk
				}
			}
		}
		if atEOF && dataleft > 0 || state != headerStart {
			err = io.ErrUnexpectedEOF
		}
		return
	}
}

// ErrBadChunk indicates a chunked framing protocol error occurred
var ErrBadChunk = errors.New("bad chunk")

var tokenEOM = []byte("]]>]]>")
