package bencoding_test

import (
	"bufio"
	"bytes"
	"slices"
	"testing"

	bencoding "github.com/nstefan13/bittorrent/bencoding"
)

func TestDecodeString(t *testing.T) {
	t.Run("decode '5:SofiaAAAAA' to 'Sofia'", func(t *testing.T) {
		buffer := newBufferedReader([]byte("5:SofiaAAAAA"))

		want := []byte("Sofia")
		got := bencoding.DecodeString(buffer)

		if !slices.Equal(got, want) {
			t.Errorf("expected %q, got %q", want, got)
		}
	})

	t.Run("decode '3:AnaAAAAA' to 'Ana'", func(t *testing.T) {
		buffer := newBufferedReader([]byte("3:AnaAAA"))

		want := []byte("Ana")
		got := bencoding.DecodeString(buffer)

		assertBuffers(t, want, got)
	})
}

func newBufferedReader(data []byte) *bufio.Reader {
	return bufio.NewReader(bytes.NewReader(data))
}

func assertBuffers(t testing.TB, expected, got []byte) {
	t.Helper()
	if !slices.Equal(got, expected) {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

// "5:"
// "5"
// "Sofia"
// "5:SofiaAAAAA"
