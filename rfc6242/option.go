package rfc6242

const (
	// DecoderMinScannerBufferSize is the scanner buffer size floor.
	DecoderMinScannerBufferSize = 20
)

// DecoderOption is a constructor option function for the Decoder type.
type DecoderOption func(*Decoder)

// WithScannerBufferSize configures the buffer size of the
// bufio.Scanner used by the decoder to scan input tokens.  If bytes
// is smaller than the constant DecoderMinScannerBufferSize, the
// buffer size will be set to DecoderMinScannerBufferSize.
func WithScannerBufferSize(bytes int) DecoderOption {
	return func(d *Decoder) {
		if bytes < DecoderMinScannerBufferSize {
			bytes = DecoderMinScannerBufferSize
		}
		d.bufSize = bytes
	}
}

// WithFramer sets the Decoder's initial Framer.
func WithFramer(f FramerFn) DecoderOption { return func(d *Decoder) { d.Framer = f } }
