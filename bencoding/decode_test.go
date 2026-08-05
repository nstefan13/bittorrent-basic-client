package bencoding_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"testing"

	bencoding "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
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

			assertNoError(t, err)
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
			_, err := bencoding.DecodeString(reader)
			want := tc.Error
			assertError(t, err, want)
		})
	}
}

func TestDecodeInt(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output bencoding.BInt
	}{
		{Input: []byte("i42e<whatever else is there>"), Output: 42},
		{Input: []byte("i244e<gibberish>"), Output: 244},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %d", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bencoding.DecodeInt(reader)

			assertNoError(t, err)
			if got != want {
				t.Errorf("expected %d, got %d", want, got)
			}
		})
	}
}

func TestDecodeIntErrors(t *testing.T) {
	testCases := []struct {
		Input []byte
		Error error
	}{
		{Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Input: []byte("i"), Error: io.ErrUnexpectedEOF},
		{Input: []byte("i23"), Error: io.ErrUnexpectedEOF},
		{Input: []byte("i-123"), Error: io.ErrUnexpectedEOF},

		{Input: []byte("X23"), Error: bencoding.ErrMalformedInput},

		{Input: []byte("ie"), Error: bencoding.ErrInvalidIntegerValue},
		{Input: []byte("i e"), Error: bencoding.ErrInvalidIntegerValue},
		{Input: []byte("i-+0e"), Error: bencoding.ErrInvalidIntegerValue},
		{Input: []byte("i-0e"), Error: bencoding.ErrInvalidIntegerValue},
		{Input: []byte("i-00e"), Error: bencoding.ErrInvalidIntegerValue},
		{Input: []byte("i-00123e"), Error: bencoding.ErrInvalidIntegerValue},
		{Input: []byte("i-123.32e"), Error: bencoding.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			_, err := bencoding.DecodeInt(reader)
			want := tc.Error
			assertError(t, err, want)
		})
	}
}

func TestDecodeAny(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output any
	}{
		{[]byte("le"), bencoding.BList{}},
		{[]byte("i42e"), bencoding.BInt(42)},

		{[]byte("0:"), bencoding.BString("")},
		{[]byte("3:Ana"), bencoding.BString("Ana")},

		{[]byte("l3:anae"), bencoding.BList{bencoding.BString("ana")}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %s", tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			got, err := bencoding.DecodeAny(reader)
			want := tc.Output

			assertNoError(t, err)

			actualTypeName := typeNameOf(got)
			expectedTypeName := typeNameOf(tc.Output)
			if actualTypeName != expectedTypeName {
				t.Fatalf("expected return value to be of type %q, insted it is of type %q", expectedTypeName, actualTypeName)
			}

			helpers.AssertEqual(t, want, got)
		})
	}
}

func TestDecodeAnyErrors(t *testing.T) {
	testCases := []struct {
		Input []byte
		Err   error
	}{
		{[]byte(""), io.ErrUnexpectedEOF},
		{[]byte("X"), bencoding.ErrMalformedInput},
		{[]byte("ie"), bencoding.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		reader := newBufferedReader(tc.Input)

		_, err := bencoding.DecodeAny(reader)
		want := tc.Err
		assertError(t, err, want)
	}
}

func TestDecodeList(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output bencoding.BList
	}{
		{
			[]byte("le"),
			[]any{},
		},
		{
			[]byte("li32e3:Anae"),
			[]any{bencoding.BInt(32), bencoding.BString("Ana")},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %d", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bencoding.DecodeList(reader)

			assertNoError(t, err)
			helpers.AssertEqual(t, want, got)
		})
	}
}

func TestDecodeListErrors(t *testing.T) {
	testCases := []struct {
		Input []byte
		Error error
	}{
		{[]byte("X"), bencoding.ErrMalformedInput},

		{[]byte(""), io.ErrUnexpectedEOF},
		{[]byte("l"), io.ErrUnexpectedEOF},
		{[]byte("li12"), io.ErrUnexpectedEOF},
		{[]byte("li12e"), io.ErrUnexpectedEOF},

		{[]byte("li12e32:e"), io.ErrUnexpectedEOF},
		{[]byte("li12e:e"), bencoding.ErrMalformedInput},
		{[]byte("li012ee"), bencoding.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			_, err := bencoding.DecodeList(reader)
			want := tc.Error
			assertError(t, err, want)
		})
	}
}

func typeNameOf(value any) string {
	return reflect.TypeOf(value).Name()
}

func newBufferedReader(data []byte) *bufio.Reader {
	return bufio.NewReader(bytes.NewReader(data))
}

func assertError(t testing.TB, err, want error) {
	if want == nil {
		panic("assertError called with `want` set to nil")
	}

	if err == nil {
		t.Fatal("expected an error to be returned")
	}

	if !errors.Is(err, want) {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

func assertNoError(t testing.TB, err error) {
	if err != nil {
		t.Fatalf("an unexpected error was returned: %q", err)
	}
}
