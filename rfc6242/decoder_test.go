package rfc6242

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var decoderCases = []struct {
	name      string
	chunked   bool
	framer    FramerFn
	input     string
	output    string
	wantError string
}{
	{
		name:    "eom/ok:empty",
		chunked: false,
		input:   "]]>]]>",
		output:  "",
	},

	{
		name:    "eom/ok:multi documents, bigger than scanner buffer",
		chunked: false,
		input:   "012345678901234567890123456789]]>]]>012345678901234567890123456789]]>]]>",
		output:  "012345678901234567890123456789012345678901234567890123456789",
	},

	{
		name:    "eom/ok:trailing character document",
		chunked: false,
		input:   "]]>]]>b]]>]]>",
		output:  "b",
	},

	{
		name:    "eom/ok:single document",
		chunked: false,
		input:   "foo]]>]]>",
		output:  "foo",
	},

	{
		name:    "eom/ok:multi documents, single character",
		chunked: false,
		input:   "]]>]]>]]>]]>]]>]]>]]>]]> ]]>]]>]]>]]>",
		output:  " ",
	},

	{
		name:    "eom/ok:two documents",
		chunked: false,
		input:   "foo]]>]]> bar]]>]]>",
		output:  "foo bar",
	},

	{
		name:    "eom/ok:incomplete eom token",
		chunked: false,
		input:   "]]>]]foo]]>]]>",
		output:  "]]>]]foo",
	},

	{
		name:    "chunked/ok:one doc two chunks",
		chunked: true,
		input:   "\n#1\nA\n#3\n AA\n##\n",
		output:  "A AA",
	},

	{
		name:    "chunked/ok:two docs",
		chunked: true,
		input:   "\n#1\nA\n#3\n AA\n##\n\n#3\nABC\n##\n",
		output:  "A AAABC",
	},

	{
		name:    "chunked/ok:three docs,one chunk",
		chunked: true,
		input:   "\n#3\nABC\n##\n\n#3\nABC\n##\n\n#3\nABC\n##\n",
		output:  "ABCABCABC",
	},

	{
		name:    "chunked/ok:three docs,multiple chunks",
		chunked: true,
		input:   "\n#3\nABC\n#3\nDEF\n##\n\n#1\n0\n#1\n0\n##\n\n#6\n/opr8t\n##\n\n#1\no\n#1\nr\n##\n",
		output:  "ABCDEF00/opr8tor",
	},

	// error cases

	{
		name:      "eom/error:empty stream",
		chunked:   false,
		input:     "",
		output:    "",
		wantError: io.ErrUnexpectedEOF.Error(),
	},

	{
		name:      "chunked/error:empty stream",
		chunked:   true,
		input:     "",
		output:    "",
		wantError: io.ErrUnexpectedEOF.Error(),
	},

	{
		name:      "chunked/error:zero chunks seen",
		chunked:   true,
		input:     "\n##\n",
		wantError: ErrZeroChunks.Error(),
	},

	{
		name:      "chunked/error:no chunk-size but other data",
		chunked:   true,
		input:     "\n#AB",
		wantError: ErrInvalidChunk.Error(),
	},

	{
		name:      "chunked/error:incomplete chunk-size",
		chunked:   true,
		input:     "\n#12345",
		wantError: io.ErrUnexpectedEOF.Error(),
	},

	{
		name:      "chunked/error:chunk-size too long",
		chunked:   true,
		input:     "\n#1234567890122\n#....",
		wantError: "token too long",
	},

	{
		name:      "chunked/error:chunk-size too big",
		chunked:   true,
		input:     "\n#4294967297\n#....",
		wantError: "bad chunk-size: chunk size larger than maximum (4294967295)",
	},

	{
		name:      "chunked/error:unexpected EOF:missing end of chunks",
		chunked:   true,
		input:     "\n#5\nABCDE",
		output:    "ABCDE", // we flush everything before detecting the error
		wantError: io.ErrUnexpectedEOF.Error(),
		// unexpected EOF due to the lack of end-of-chunks to
		// terminate the stream.
	},

	{
		name:      "chunked/error:three docs,multiple chunks broken at the XX",
		chunked:   true,
		input:     "\n#3\nABC\n#3\nDEF\n##\n\n#1\n0\n#1\n0\n##\n\n#6\n/opr8tXX\n##\n\n#1\no\n#1\nr\n##\n",
		wantError: "protocol error: invalid chunk header",
		output:    "ABCDEF00/opr8t", // confirm we flush up to end of the valid token
	},

	{
		name:      "chunked/error:unexpected EOF:long chunk",
		chunked:   true,
		input:     "\n#4\nABCDE",
		output:    "ABCD",
		wantError: "protocol error: invalid chunk header",
	},

	// rfc6242 test cases

	{
		name:    "chunked/ok:RFC6242 s4.2",
		chunked: true,
		input:   "\n#4\n<rpc\n#18\n message-id=\"102\"\n\n##\n\n#79\n     xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <close-session/>\n</rpc>\n##\n",
		output:  "<rpc message-id=\"102\"\n     xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <close-session/>\n</rpc>",
	},

	{
		name:    "chunked/ok:RFC6242 s5:C",
		chunked: true,
		input:   "\n#140\n<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rpc message-id=\"106\"\n     xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <close-session/>\n</rpc>\n##\n",
		output:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rpc message-id=\"106\"\n     xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <close-session/>\n</rpc>",
	},

	{
		name:    "chunked/ok:RFC6242 s5:S",
		chunked: true,
		input:   "\n#139\n<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rpc-reply id=\"106\"\n           xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <ok/>\n</rpc-reply>\n##\n",
		output:  "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rpc-reply id=\"106\"\n           xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <ok/>\n</rpc-reply>",
	},
}

func testDecoderGetDecoder(chunked bool, input string) *Decoder {
	var framer FramerFn
	if chunked {
		framer = DecoderChunked
	} else {
		framer = DecoderEndOfMessage
	}
	return NewDecoder(
		strings.NewReader(input),
		WithFramer(framer),
		WithScannerBufferSize(DecoderMinScannerBufferSize))
}

func TestDecoderWriterTo(t *testing.T) {
	for _, tc := range decoderCases {
		t.Run(tc.name,
			func(t *testing.T) {
				d := testDecoderGetDecoder(tc.chunked, tc.input)
				ck := assert.New(t)
				// io.Copy will use d's io.WriterTo implementation.
				output := &bytes.Buffer{}
				n, err := io.Copy(output, d)
				var errMsg string
				if err != nil {
					errMsg = err.Error()
				}
				if !ck.Contains(errMsg, tc.wantError) && err != nil {
					t.Logf("error was: %+v", err)
				}
				if tc.wantError == "" {
					ck.NoErrorf(err, "wanted no error but got: %v", err)
				}
				outputMsg := output.String()
				ck.Equal(int64(len(outputMsg)), n)
				ck.Equal(tc.output, outputMsg)
			})
	}
}

// decoderReaderOnly is a Decoder masked to hide its io.WriterTo
// implementation (so io.Copy uses its Read() function).
type decoderReaderOnly struct{ d *Decoder }

func (d decoderReaderOnly) Read(b []byte) (int, error) { return d.d.Read(b) }

func TestDecoderReader(t *testing.T) {
	for _, tc := range decoderCases {
		t.Run(tc.name,
			func(t *testing.T) {
				// strip the WriteTo method from the original decoder object,
				// to force the io.Reader implementation to be used by io.Copy.
				d := decoderReaderOnly{testDecoderGetDecoder(tc.chunked, tc.input)}
				ck := assert.New(t)
				output := &bytes.Buffer{}
				// io.Copy uses the io.Reader implementation, which
				// uses a background goroutine to push filtered data
				// through a pipe.
				n, err := io.Copy(output, d)
				var errMsg string
				if err != nil {
					errMsg = err.Error()
				}
				if !ck.Contains(errMsg, tc.wantError) && err != nil {
					t.Logf("error was: %+v", err)
				}
				if tc.wantError == "" {
					ck.NoErrorf(err, "wanted no error but got: %v", err)
				}
				outputMsg := output.String()
				ck.Equal(int64(len(outputMsg)), n)
				ck.Equal(tc.output, outputMsg)
			})
	}
}

type devNull struct{}

func (n devNull) Write(b []byte) (int, error) { return len(b), nil }

var benchmarkData = map[string][]byte{
	"eom/hello": []byte(`<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
<capabilities>
  <capability>
    urn:ietf:params:netconf:base:1.1
  </capability>
  <capability>
    urn:ietf:params:ns:netconf:capability:startup:1.0
  </capability>
</capabilities>
<session-id>4</session-id>
</hello>`),
	"chunked/rfc6242-s5": []byte("\n#139\n<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<rpc-reply id=\"106\"\n           xmlns=\"urn:ietf:params:xml:ns:netconf:base:1.0\">\n  <ok/>\n</rpc-reply>\n##\n"),
}

func BenchmarkDecoderEndOfMessageReader(b *testing.B) {
	benchmarkDecoderReader(b, benchmarkData["eom/hello"], DecoderEndOfMessage)
}

func BenchmarkDecoderEndOfMessageWriterTo(b *testing.B) {
	benchmarkDecoderWriterTo(b, benchmarkData["eom/hello"], DecoderEndOfMessage)
}

func BenchmarkDecoderChunkedWriterTo(b *testing.B) {
	benchmarkDecoderWriterTo(b, benchmarkData["chunked/rfc6242-s5"], DecoderChunked)
}

func BenchmarkDecoderChunkedReader(b *testing.B) {
	benchmarkDecoderReader(b, benchmarkData["chunked/rfc6242-s5"], DecoderChunked)
}

func benchmarkDecoderWriterTo(b *testing.B, msg []byte, framer FramerFn) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := NewDecoder(bytes.NewReader(msg), WithFramer(DecoderEndOfMessage))
		io.Copy(devNull{}, d)
	}
}

func benchmarkDecoderReader(b *testing.B, msg []byte, framer FramerFn) {
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		d := decoderReaderOnly{NewDecoder(bytes.NewReader(msg), WithFramer(DecoderEndOfMessage))}
		io.Copy(devNull{}, d)
	}
}
