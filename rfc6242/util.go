// Copyright (c) 2018 Andrew Fort
//

package rfc6242

// SetChunkedFraming enables chunked framing mode on any non-nil
// *Decoder and *Encoder objects passed to it.
func SetChunkedFraming(objects ...interface{}) {
	for _, obj := range objects {
		switch obj := obj.(type) {
		case *Decoder:
			if obj != nil {
				obj.framer = decoderChunked
			}
		case *Encoder:
			if obj != nil {
				obj.ChunkedFraming = true
			}
		}
	}
}

// ClearChunkedFraming disables chunked framing mode on any non-nil
// *Decoder and *Encoder objects passed to it.
func ClearChunkedFraming(objects ...interface{}) {
	for _, obj := range objects {
		switch obj := obj.(type) {
		case *Decoder:
			if obj != nil {
				obj.framer = decoderEndOfMessage
			}
		case *Encoder:
			if obj != nil {
				obj.ChunkedFraming = false
			}
		}
	}
}
