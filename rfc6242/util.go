// Copyright (c) 2018 Andrew Fort
//

package rfc6242

// SetChunkedFraming enables chunked framing mode on all
// *Decoder and *Encoder objects passed.
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
