package transport

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWriter(t *testing.T) {
	for _, tc := range []struct {
		f func(*assert.Assertions)
	}{
		{
			f: func(a *assert.Assertions) {
				b := closeBuffer{&bytes.Buffer{}}
				w := NewWriter(b)
				w.Write([]byte("foo"))
				w.WriteEnd()
				a.Equal("foo]]>]]>", b.String())
			},
		},
		{
			f: func(a *assert.Assertions) {
				b := closeBuffer{&bytes.Buffer{}}
				w := NewWriter(b)
				w.SetFramingMode(true)
				w.Write([]byte("foo"))
				w.WriteEnd()
				a.Equal("\n#3\nfoo\n##\n", b.String())
			},
		},
		{
			f: func(a *assert.Assertions) {
				b := closeBuffer{&bytes.Buffer{}}
				w := NewWriter(b)
				w.Write([]byte("foo"))
				w.WriteEnd()
				w.SetFramingMode(true)
				w.Write([]byte("b"))
				w.Write([]byte("ar"))
				w.Write([]byte("baz"))
				w.WriteEnd()
				a.Equal("foo]]>]]>\n#1\nb\n#2\nar\n#3\nbaz\n##\n", b.String())
			},
		},
	} {
		t.Run("", func(t *testing.T) { tc.f(assert.New(t)) })
	}
}

type closeBuffer struct{ *bytes.Buffer }

func (cb closeBuffer) Close() error { return nil }
