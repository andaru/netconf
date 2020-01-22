package transport

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	for _, tc := range []struct {
		in string
		f  func(*assert.Assertions)
	}{
		{
			f: func(a *assert.Assertions) {
				r := strings.NewReader("foo]]>]]>bar]]>]]>baz]]>]]>")
				var gotEOM int
				rdr := NewReader(r, func() { gotEOM++ })
				b := closeBuffer{&bytes.Buffer{}}
				n, err := io.Copy(b, rdr)
				a.NoError(err)
				a.Greater(n, int64(0))
				a.Equal(3, gotEOM)
				a.Equal("foobarbaz", b.String())
			},
		},
		{
			f: func(a *assert.Assertions) {
				r := strings.NewReader("foo]]>]]>\n#3\nbar\n##\n")
				var gotEOM int
				var rdr *Reader
				rdr = NewReader(r, func() {
					if gotEOM == 0 {
						rdr.SetFramingMode(true)
					}
					gotEOM++
				})
				b := closeBuffer{&bytes.Buffer{}}
				n, err := io.Copy(b, rdr)
				a.NoError(err)
				a.Greater(n, int64(0))
				a.Equal(2, gotEOM)
				a.Equal("foobar", b.String())
			},
		},
		// {
		// 	f: func(a *assert.Assertions) {
		// 		b := closeBuffer{&bytes.Buffer{}}
		// 		w := NewWriter(b)
		// 		w.SetFramingMode(true)
		// 		w.Write([]byte("foo"))
		// 		w.WriteEnd()
		// 		a.Equal("\n#3\nfoo\n##\n", b.String())
		// 	},
		// },
		// {
		// 	f: func(a *assert.Assertions) {
		// 		b := closeBuffer{&bytes.Buffer{}}
		// 		w := NewWriter(b)
		// 		w.Write([]byte("foo"))
		// 		w.WriteEnd()
		// 		w.SetFramingMode(true)
		// 		w.Write([]byte("b"))
		// 		w.Write([]byte("ar"))
		// 		w.Write([]byte("baz"))
		// 		w.WriteEnd()
		// 		a.Equal("foo]]>]]>\n#1\nb\n#2\nar\n#3\nbaz\n##\n", b.String())
		// 	},
		// },
	} {
		t.Run("", func(t *testing.T) { tc.f(assert.New(t)) })
	}
}
