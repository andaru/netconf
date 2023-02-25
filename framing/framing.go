package framing

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"bufio"
)

type ErrBadChunk struct {
	Message string
	Offset  int
}

func (e ErrBadChunk) Error() string {
	msg := "netconf bad chunk"
	if e.Message != "" {
		msg = msg + ": " + e.Message
	}
	if e.Offset < 1 {
		return msg
	}
	return fmt.Sprintf("%s at input offset %d", msg, e.Offset)
}

// tokenEOM is the message termination token found in end-of-message encoding streams
var tokenEOM = []byte("]]>]]>")

// SplitEOM returns a bufio.SplitFunc suitable for RFC6242
// "end-of-message delimited" NETCONF transport streams.
//
// endOfMessage will be called at the end of each NETCONF message.
func SplitEOM(endOfMessage func()) bufio.SplitFunc {
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
			if endOfMessage != nil {
				endOfMessage()
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
// endOfMessage will be called at the end of each NETCONF message.
//
// It must only be used with bufio.Scanner who have a buffer of
// at least 16 bytes.
func SplitChunked(endOfMessage func()) bufio.SplitFunc {
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
			// Each chunk header is at least 4 bytes, so ask for at least that
			// (unless we're at EOF, in which case we check length again later)
			if len(cur) < 4 && !atEOF {
				return
			}
			// chunked message decoding state machine
			switch state {
			case headerStart: // initial state/start of message header
				switch {
				case bytes.HasPrefix(cur, []byte("\n#")): // expected chunked message intro
					if len(cur) < 4 { // we need at least 4 bytes for a valid header (let alone data body)
						err = ErrBadChunk{Message: "truncated chunk header"}
						return
					}
					switch r := cur[2]; {
					case r == '#': // end of message indicator
						advance += 3
						state = endOfChunks
					case r >= '1' && r <= '9': // chunk size: starts with a 1-9 character
						advance += 2
						state = headerSize // move onto decoding the header size
					default:
						err = ErrBadChunk{Message: "invalid chunk size"}
					}
				default:
					err = ErrBadChunk{Message: "invalid chunk header"}
				}
			case headerSize: // decode the chunk length
				switch idx := bytes.IndexByte(cur, '\n'); {
				case idx < 1, idx > 10:
					// we store chunk size as an int handled as a 32 bit int, hence 10 character max
					if len(cur) < 11 && !atEOF {
						// ask for more data (happens in the idx == -1 case only due the guard)
						return
					} else if idx > 10 {
						err = ErrBadChunk{Message: "chunk size too large"}
					} else {
						err = ErrBadChunk{Message: "invalid chunk size"}
					}
				default:
					csize := cur[:idx]
					if csizeVal, csizeErr := strconv.ParseUint(string(csize), 10, 31); csizeErr != nil {
						seMsg := csizeErr.Error()
						if se, _ := csizeErr.(*strconv.NumError); se != nil {
							seMsg = strings.TrimSpace(strings.Join(strings.Split(seMsg, ":")[1:], ":"))
						}
						err = ErrBadChunk{Message: "invalid chunk size: " + seMsg}
					} else {
						advance += idx + 1
						dataleft = int(csizeVal)
						state = data
					}
				}
			case data: // extract the message data
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
			case endOfChunks: // end of message reached
				switch r := cur[0]; {
				case r == '\n' && cs > 0: // expect chunk footer and that we've seen at least one chunk
					advance++
					state = headerStart
					if endOfMessage != nil {
						endOfMessage()
					}
				default:
					err = ErrBadChunk{Message: "invalid chunk terminator"}
				}
			}
		}
		// catch unexpected EOF conditions
		if err == nil && (atEOF && dataleft > 0 || state != headerStart) {
			err = io.ErrUnexpectedEOF
		}
		return
	}
}
