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
//
// It must only be used with bufio.Scanner who have a buffer of
// at least 6 bytes (rarely is this a concern).
func SplitEOM(cb func()) bufio.SplitFunc {
	var eofOK, seen bool
	return func(b []byte, atEOF bool) (advance int, token []byte, err error) {
		// fmt.Printf("> b=%q atEOF=%v (advance=%d token=%q err=%v)\n", b, atEOF, advance, token, err)
		// defer func() { fmt.Printf("< b=%q atEOF=%v (advance=%d token=%q err=%v)\n", b, atEOF, advance, token, err) }()

		if atEOF && len(b) == 0 && !eofOK && seen {
			err = io.ErrUnexpectedEOF
			return
		}
		seen = seen || len(b) > 0
		for cur := b[advance:]; advance < len(b); cur = b[advance:] {
			if len(cur) < len(tokenEOM) && !atEOF {
				return
			}
			switch idxeom := bytes.IndexByte(cur, ']'); {
			case idxeom == -1:
				advance += len(cur)
				token = append(token, cur...)
			case idxeom > 0:
				advance += idxeom
				token = append(token, cur[:idxeom]...)
				return
			case idxeom == 0:
				advance++
				for i := 1; i < len(tokenEOM); advance, i = advance+1, i+1 {
					if cur[i] != tokenEOM[i] {
						token = append(token, cur[:i]...)
						return
					}
				}
				if cb != nil {
					cb()
				}
				if !atEOF {
					eofOK = bytes.HasSuffix(b, tokenEOM)
					return
				}
			}
		}
		eofOK = bytes.HasSuffix(b, tokenEOM)
		return
	}
}

// SplitChunked returns a bufio.SplitFunc suitable for decoding
// "chunked framing" NETCONF transport streams.
//
// It must only be used with bufio.Scanner who have a buffer of
// at least 16 bytes (rarely is this a concern).
func SplitChunked() bufio.SplitFunc {
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
		// fmt.Printf("+> b=%q atEOF=%v (advance=%d token=%q err=%v)\n", b, atEOF, advance, token, err)
		// defer func() { fmt.Printf("+> b=%q atEOF=%v (advance=%d token=%q err=%v)\n", b, atEOF, advance, token, err) }()
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
