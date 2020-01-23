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
	} {
		t.Run("", func(t *testing.T) { tc.f(assert.New(t)) })
	}
}

func TestSetFramingModeDouble(t *testing.T) {
	a := assert.New(t)
	r := NewReader(strings.NewReader(""), func() {})
	r.SetFramingMode(true)
	a.Panics(func() { r.SetFramingMode(true) })
}
