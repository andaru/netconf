package rfc6242

// SetChunkedFraming enables chunked framing mode on all
// *Decoder and *Encoder objects passed.
func SetChunkedFraming(objects ...interface{}) {
	for _, obj := range objects {
		switch obj := obj.(type) {
		case *Decoder:
			obj.Framer = decoderChunked
		case *Encoder:
			obj.ChunkedFraming = true
		}
	}
}
