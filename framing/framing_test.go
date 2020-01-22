package framing

import (
	"fmt"
	"testing"

	"bufio"

	"strings"

	"github.com/stretchr/testify/assert"
)

func TestFramingEOM(t *testing.T) {
	for _, tc := range []struct {
		input  string
		want   string
		hasErr bool
		wantCB int
	}{
		{input: "]]>]]>", want: "", wantCB: 1},
		{
			input:  "012345789012345789012345789012345789012345789012345789012345789]]>]]>012345789012345789012345789]]>]]>",
			want:   "012345789012345789012345789012345789012345789012345789012345789012345789012345789012345789",
			wantCB: 2,
		},
		{
			input:  "0123456789]]>]]>0123456789]]>]]>0123456789]]>]]>0123456789]]>]]>]]>]]>0123456789]]>]]>]]>]]>",
			want:   "01234567890123456789012345678901234567890123456789",
			wantCB: 7,
		},
		{
			input:  "]]>]]foo]]>]]>bar]]]]]>]]>]]]>]]]]>]]>baz]]>]]>",
			want:   "]]>]]foobar]]]]]]>]]baz",
			wantCB: 4, // should be 4, no?
		},
		{
			input:  "]]>]]foo]]>]]>bar]]]>]]>",
			want:   "]]>]]foobar]",
			wantCB: 2,
		},
		{input: "foo>]]>bar]]>]]>bazoopa]]>]]>", want: "foo>]]>barbazoopa", wantCB: 2},
		{input: "]]>]]>]]>]]>baz]]>]]>", want: "baz", wantCB: 3},
		{input: "]]>]]>]]>]]>]]>]]>", want: "", wantCB: 3},
		{input: "]]>]]>foo]]>]]>]]>]]>", want: "foo", wantCB: 3},
		{input: "]]>]]>foo\n]]>]]>]]>]]>bar]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>", want: "foo\nbar", wantCB: 11},
		{input: "foo]]>]]>", want: "foo", wantCB: 1},
		{input: "foo]]>]]>bar]]>]]>bazoopa]]>]]>", want: "foobarbazoopa", wantCB: 3},
		{input: "foo", want: "foo", hasErr: true},
		{input: "foo]]>]]>bar]]>]]>bazoopa", want: "foobarbazoopa", hasErr: true, wantCB: 2},
		{input: "]]>]]>]]>]]>]]>]]>abcdefghijklmnop", want: "abcdefghijklmnop", hasErr: true, wantCB: 3},
		{input: "a]]>]]>b]]>]]>c", want: "abc", hasErr: true, wantCB: 2},
		{},
	} {
		for bsize := 16; bsize < 65; bsize++ {
			t.Run(fmt.Sprintf("%s/%d", tc.input, bsize), func(t *testing.T) {
				ck := assert.New(t)
				scanner := bufio.NewScanner(strings.NewReader(tc.input))
				scanner.Buffer(make([]byte, bsize), bsize*4)
				var gotCB int
				scanner.Split(SplitEOM(func() { gotCB++ }))
				var got string
				for scanner.Scan() {
					got += scanner.Text()
				}
				serr := scanner.Err()
				ck.True(serr == nil && !tc.hasErr || serr != nil && tc.hasErr, "want an error only if hasErr true, got %v (hasErr %v)", serr, tc.hasErr)
				ck.Equal(tc.want, got)
				ck.Equal(tc.wantCB, gotCB)
			})
		}
	}
}

func BenchmarkFramingEOM(b *testing.B) {
	for _, tc := range []struct {
		input  string
		want   string
		hasErr bool
	}{
		{input: "]]>]]>", want: ""},
		{
			input: "012345789012345789012345789012345789012345789012345789012345789]]>]]>012345789012345789012345789]]>]]>",
			want:  "012345789012345789012345789012345789012345789012345789012345789012345789012345789012345789",
		},
		{
			input: "0123456789]]>]]>0123456789]]>]]>0123456789]]>]]>0123456789]]>]]>]]>]]>0123456789]]>]]>]]>]]>",
			want:  "01234567890123456789012345678901234567890123456789",
		},
		{
			input: "]]>]]foo]]>]]>bar]]]]]>]]>]]]>]]]]>]]>baz]]>]]>",
			want:  "]]>]]foobar]]]]]>]]>]]]>]]baz",
		},
		{input: "foo>]]>bar]]>]]>bazoopa]]>]]>", want: "foo>]]>barbazoopa"},
		{input: "]]>]]>]]>]]>baz]]>]]>", want: "baz"},
		{input: "]]>]]>]]>]]>]]>]]>", want: ""},
		{input: "]]>]]>foo]]>]]>]]>]]>", want: "foo"},
		{input: "]]>]]>foo\n]]>]]>]]>]]>bar]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>]]>", want: "foo\nbar"},
		{input: "foo]]>]]>", want: "foo"},
		{input: "foo]]>]]>bar]]>]]>bazoopa]]>]]>", want: "foobarbazoopa"},
		{input: "foo", want: "foo", hasErr: true},
		{input: "foo]]>]]>bar]]>]]>bazoopa", want: "foobarbazoopa", hasErr: true},
		{input: "]]>]]>]]>]]>]]>]]>abcdefghijklmnop", want: "abcdefghijklmnop", hasErr: true},
		{input: "a]]>]]>b]]>]]>c", want: "abc", hasErr: true},
	} {
		b.Run(tc.input, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				scanner := bufio.NewScanner(strings.NewReader(tc.input))
				scanner.Buffer(make([]byte, 64), 64)
				scanner.Split(SplitEOM(nil))
				for scanner.Scan() {
				}
			}
		})
	}
}

func TestFramingChunked(t *testing.T) {
	for _, tc := range []struct {
		input  string
		want   string
		hasErr bool
	}{
		// empty input (special case, no end-of-chunks required)
		{input: "", want: "", hasErr: false},
		// valid data
		{input: "\n#1\na\n##\n", want: "a"},
		{input: "\n#1\na\n#1\nb\n#1\nc\n##\n", want: "abc"},
		{input: "\n#2\nab\n#2\ncd\n#2\nef\n##\n", want: "abcdef"},
		{input: "\n#3\nfoo\n#4\nfood\n##\n", want: "foofood"},
		{input: "\n#4\nabc\n\n#4\ndef\n\n##\n", want: "abc\ndef\n"},

		// coverage of all error causes
		{input: "\n##\n", want: "", hasErr: true},
		{input: "foo]]>]]>bar]]>]]>baz", want: "", hasErr: true},
		{input: "\n#03\nfoo\n##\n", hasErr: true},
		{input: "\n#92147483648\nffffffff...", hasErr: true},
		{input: "\n#9\n012", hasErr: true},
		{input: "\n#\na\n##\n", hasErr: true},
		{input: "\n#1a\na\n##\n", hasErr: true},
		{input: "\n#1\na\n##", hasErr: true},
		{input: "\n#1\na\n#\n ", hasErr: true},
		{input: "\n#1\na\n#", hasErr: true},
		{input: "\n#1\na\n##\n ", hasErr: true},
		{input: "\n#9\n0123456789\n##\n", hasErr: true},
	} {
		minlen := 16
		max := 32
		for bsize := minlen; bsize < (minlen+1)+max; bsize++ {
			t.Run(fmt.Sprintf("%s/%d", tc.input, bsize), func(t *testing.T) {
				ck := assert.New(t)
				rdr := strings.NewReader(tc.input)
				scanner := bufio.NewScanner(rdr)
				scanner.Buffer(make([]byte, bsize), bsize*2)
				scanner.Split(SplitChunked(nil))
				var got string
				for scanner.Scan() {
					got += scanner.Text()
				}
				serr := scanner.Err()
				ck.True(serr == nil && !tc.hasErr || serr != nil && tc.hasErr, "want an error only if hasErr true, got %v (hasErr %v)", serr, tc.hasErr)
				if tc.want != "" {
					ck.Equal(tc.want, got)
				}
			})
		}
	}
}

func BenchmarkFramingChunked(b *testing.B) {
	for _, tc := range []struct {
		input  string
		want   string
		hasErr bool
	}{
		// empty input (special case, no end-of-chunks required)
		{input: "", want: "", hasErr: false},
		// valid data
		{input: "\n#1\na\n##\n", want: "a"},
		{input: "\n#1\na\n#1\nb\n#1\nc\n##\n", want: "abc"},
		{input: "\n#2\nab\n#2\ncd\n#2\nef\n##\n", want: "abcdef"},
		{input: "\n#3\nfoo\n#4\nfood\n##\n", want: "foofood"},
		{input: "\n#4\nabc\n\n#4\ndef\n\n##\n", want: "abc\ndef\n"},

		// coverage of all error causes
		{input: "\n##\n", want: "", hasErr: true},
		{input: "foo]]>]]>bar]]>]]>baz", want: "", hasErr: true},
		{input: "\n#03\nfoo\n##\n", hasErr: true},
		{input: "\n#92147483648\nffffffff...", hasErr: true},
		{input: "\n#9\n012", hasErr: true},
		{input: "\n#\na\n##\n", hasErr: true},
		{input: "\n#1a\na\n##\n", hasErr: true},
		{input: "\n#1\na\n##", hasErr: true},
		{input: "\n#1\na\n#\n ", hasErr: true},
		{input: "\n#1\na\n#", hasErr: true},
		{input: "\n#1\na\n##\n ", hasErr: true},
		{input: "\n#9\n0123456789\n##\n", hasErr: true},
	} {
		b.Run(tc.input, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				scanner := bufio.NewScanner(strings.NewReader(tc.input))
				scanner.Buffer(make([]byte, 64), 64)
				scanner.Split(SplitChunked(nil))
				for scanner.Scan() {
				}
			}
		})
	}
}
