package rfc6242

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeEncode(t *testing.T) {
	for _, tc := range decoderCases {
		if tc.wantError != "" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			ck := assert.New(t)
			d := testDecoderGetDecoder(tc.chunked, tc.input)
			output := new(bytes.Buffer)
			e := NewEncoder(output)
			n, err := io.Copy(e, d)
			ck.NoError(err)
			ck.Equal(int64(len(tc.output)), n)
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
