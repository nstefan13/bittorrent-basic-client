package bencoding_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"testing"

	bencoding "github.com/nstefan13/bittorrent/bencoding"
)

func TestDecodeString(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output []byte
	}{
		{Input: []byte("5:SofiaAAA"), Output: []byte("Sofia")},
		{Input: []byte("3:AnaASASlkDJLJASK"), Output: []byte("Ana")},
		{Input: []byte("0:"), Output: []byte("")},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %q", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bencoding.DecodeString(reader)

			if err != nil {
				t.Fatalf("an unexpected error was returned: %q", err)
			}

			if !slices.Equal(got, want) {
				t.Errorf("expected %q, got %q", want, got)
			}
		})
	}
}

func TestDecodeStringErrors(t *testing.T) {
	testCases := []struct {
		Input []byte
		Error error
	}{
		{Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Input: []byte("5"), Error: io.ErrUnexpectedEOF},
		{Input: []byte("Sofia"), Error: io.ErrUnexpectedEOF},
		{Input: []byte("5:"), Error: io.ErrUnexpectedEOF},
		{Input: []byte("10000:too_small_of_a_string"), Error: io.ErrUnexpectedEOF},
		{Input: []byte(":"), Error: bencoding.ErrInvalidStringLength},
		{Input: []byte("-1:"), Error: bencoding.ErrInvalidStringLength},
		{Input: []byte(" 5:Sofia"), Error: bencoding.ErrInvalidStringLength},
		{Input: []byte("5 :Sofia"), Error: bencoding.ErrInvalidStringLength},
		{Input: []byte("17179869184:Sofia"), Error: bencoding.ErrInvalidStringLength},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			want := tc.Error
			_, err := bencoding.DecodeString(reader)

			if err == nil {
				t.Fatal("expected an error to be returned")
			}

			if !errors.Is(err, want) {
				t.Errorf("expected %q, got %q", want, err.Error())
			}
		})
	}
}

func newBufferedReader(data []byte) *bufio.Reader {
	return bufio.NewReader(bytes.NewReader(data))
}
