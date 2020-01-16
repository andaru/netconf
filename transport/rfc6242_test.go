package transport

import (
	"fmt"
	"strings"
	"testing"

	"bytes"

	"encoding/xml"

	"github.com/stretchr/testify/assert"
)

func TestRFC6242Decoder(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		min  int
	}{
		{
			min:  6,
			name: "end-of-message-mode-only",
			in: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
	<capabilities>
		<capability>
			urn:ietf:params:netconf:base:1.0
		</capability>
	</capabilities>
</hello>]]>]]>
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
]]>]]>`,
		},
		{
			name: "chunked-switch",
			in: `
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
	<capabilities>
		<capability>
			urn:ietf:params:netconf:base:1.1
		</capability>
		<capability>
			urn:ietf:params:ns:netconf:capability:startup:1.0
		</capability>
	</capabilities>
</hello>]]>]]>
#59
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0"></rpc>
##
`,
		},
	} {
		bufsizes := []int{16, 31, 32, 64, 128, 4096, 16384, 32768, 32767, 65535, 65536, 131072}

		for i := 0; i < len(bufsizes); i++ {
			t.Run(fmt.Sprintf("%s/%d", tc.name, bufsizes[i]), func(t *testing.T) {
				a := assert.New(t)
				rdr := strings.NewReader(tc.in)

				// framing mode switch test setup
				var useBase11 bool
				// Explicitly define the Decoder reference so we can refer to its fields
				// in our callback which is defined along with the decoder itself
				var d *Decoder
				d = NewDecoder(rdr, WithDecoderBufferSize(bufsizes[i]), WithEOMCallback(func() { d.SetFramingMode(useBase11) }))

				xd := xml.NewDecoder(d)
				var err error
				var tok xml.Token
				var matchCData bool
				var tokens []xml.Token

				// emulate the capabilities exchange part of a NETCONF
				// including the optional receiver framing mode switchover
				for err == nil {
					switch tok, err = xd.Token(); {
					case err != nil && tok == nil:
					case tok != nil && err == nil:
						switch tok := tok.(type) {
						case xml.StartElement:
							// enable CData matching for <capability> elements
							matchCData = tok.Name.Local == "capability"
							if tok.Name.Local == "rpc" {
								tokens = append(tokens, tok)
							}
						case xml.EndElement:
							if tok.Name.Local == "rpc" {
								tokens = append(tokens, tok)
							}
						case xml.CharData:
							if matchCData {
								matchCData = false
								switch tr := bytes.TrimSpace(tok); string(tr) {
								case "urn:ietf:params:netconf:base:1.1":
									useBase11 = true
								}
							}
						}
					default:
						panic(fmt.Errorf("got err=%v || tok=%v", err, tok))
					}
				}
				// always expect a regular EOF at end of input
				a.EqualError(err, "EOF")
				// always expect an <rpc> and </rpc> element
				if a.Equal(2, len(tokens)) {
					t1, ok := tokens[0].(xml.StartElement)
					if a.True(ok) {
						a.Equal("rpc", t1.Name.Local)
					}
					t2, ok := tokens[1].(xml.EndElement)
					if a.True(ok) {
						a.Equal("rpc", t2.Name.Local)
					}

				}
			})
		}
	}
}

func TestRFC6242Encoder(t *testing.T) {
	for _, tc := range []struct {
		f    func(*assert.Assertions) *bytes.Buffer
		want string
	}{
		{
			f: func(ck *assert.Assertions) (b *bytes.Buffer) {
				b = &bytes.Buffer{}
				e := NewEncoder(b)
				n, err := e.Write([]byte("foo"))
				ck.Equal(3, n)
				ck.NoError(err)
				n, err = e.WriteEnd()
				ck.Equal(6, n)
				ck.NoError(err)
				return b
			},
			want: "foo]]>]]>",
		},
		{
			f: func(ck *assert.Assertions) (b *bytes.Buffer) {
				b = &bytes.Buffer{}
				e := NewEncoder(b)
				n, err := e.Write([]byte("foo"))
				ck.Equal(3, n)
				ck.NoError(err)
				n, err = e.WriteEnd()
				ck.Equal(6, n)
				ck.NoError(err)
				e.SetChunkedFraming()
				n, err = e.Write([]byte("foo"))
				ck.Equal(2+1+1+3, n)
				ck.NoError(err)
				n, err = e.WriteEnd()
				ck.Equal(4, n)
				ck.NoError(err)
				return b
			},
			want: "foo]]>]]>\n#3\nfoo\n##\n",
		},
	} {
		t.Run("want="+tc.want, func(t *testing.T) {
			ck := assert.New(t)
			buf := tc.f(ck)
			ck.Equal(tc.want, buf.String())
		})
	}
}
