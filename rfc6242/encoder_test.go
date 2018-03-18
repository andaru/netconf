package rfc6242

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncoder(t *testing.T) {
	for _, tc := range []struct {
		chunked      bool
		maxChunkSize int
		input        []string
		output       string
	}{
		{
			chunked: true,
			input:   []string{"foo"},
			output:  "\n#3\nfoo\n##\n",
		},

		{
			chunked: false,
			input:   []string{"foo"},
			output:  "foo]]>]]>",
		},

		{
			chunked: true,
			input:   []string{"foo", "foobar"},
			output:  "\n#3\nfoo\n##\n\n#6\nfoobar\n##\n",
		},

		{
			chunked: false,
			input:   []string{"foo", "foobar"},
			output:  "foo]]>]]>foobar]]>]]>",
		},
	} {
		t.Run(fmt.Sprintf("chunked=%v:%q", tc.chunked, tc.input), func(t *testing.T) {
			ck := assert.New(t)
			output := &bytes.Buffer{}
			e := NewEncoder(output)
			e.ChunkedFraming = tc.chunked
			for _, line := range tc.input {
				wn, err := e.Write([]byte(line))
				ck.NoError(err)
				ck.NoError(e.EndOfMessage())
				ck.Equal(len(line), wn)
			}
			ck.Equal(tc.output, output.String())
		})

	}
}

func TestEncoderEndOfMessage(t *testing.T) {
	ck := assert.New(t)
	output := new(bytes.Buffer)
	e := NewEncoder(output)
	e.ChunkedFraming = false
	e.EndOfMessage()
	ck.Equal("]]>]]>", output.String())

	output.Reset()
	e.ChunkedFraming = true
	e.EndOfMessage()
	ck.Equal("\n##\n", output.String())
}

type testWriterCloser struct{ calledWrite, calledClose bool }

func (wc *testWriterCloser) Write(b []byte) (int, error) {
	wc.calledWrite = true
	return len(b), nil
}

func (wc *testWriterCloser) Close() error {
	wc.calledClose = true
	return nil
}

func TestEncoderClose(t *testing.T) {
	ck := assert.New(t)
	output := &testWriterCloser{}
	e := NewEncoder(output)
	e.Write([]byte("."))
	ck.NoError(e.Close())
	ck.True(output.calledWrite)
	ck.True(output.calledClose)
}

func TestEncoderWithMaximumChunkSize(t *testing.T) {
	for _, tc := range []struct {
		size     uint32
		wantSize uint32
		input    string
		output   string
	}{
		{
			size:     0,
			wantSize: rfc6242maximumAllowedChunkSize,
		},
		{
			size:     0,
			wantSize: rfc6242maximumAllowedChunkSize,
			input:    "01234",
			output:   "\n#5\n01234",
		},
		{
			size:   4,
			input:  "012345678",
			output: "\n#4\n0123\n#4\n4567\n#1\n8",
		},
		{
			size:   1,
			input:  "012",
			output: "\n#1\n0\n#1\n1\n#1\n2",
		},
	} {
		t.Run(fmt.Sprintf("size=%d/input=%q", tc.size, tc.input), func(t *testing.T) {
			ck := assert.New(t)
			output := new(bytes.Buffer)
			e := NewEncoder(output, WithMaximumChunkSize(tc.size))
			e.ChunkedFraming = true
			e.Write([]byte(tc.input))
			if tc.wantSize != 0 {
				ck.Equal(tc.wantSize, e.MaxChunkSize)
			} else {
				ck.Equal(tc.size, e.MaxChunkSize)
			}

			ck.Equal(tc.output, output.String())
		})
	}
}
