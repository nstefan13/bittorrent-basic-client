package bencoding_test

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/nstefan13/bittorrent/bencoding"
	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
)

func TestDecodeString(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output bcd.BString
	}{
		{Input: []byte("5:SofiaAAA"), Output: bcd.BString("Sofia")},
		{Input: []byte("3:AnaASASlkDJLJASK"), Output: bcd.BString("Ana")},
		{Input: []byte("0:"), Output: bcd.BString("")},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %q", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bcd.DecodeString(reader)

			assertNoError(t, err)
			helpers.AssertEqual(t, got, want)
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
		{Input: []byte(":"), Error: bcd.ErrInvalidStringLength},
		{Input: []byte("-1:"), Error: bcd.ErrInvalidStringLength},
		{Input: []byte(" 5:Sofia"), Error: bcd.ErrInvalidStringLength},
		{Input: []byte("5 :Sofia"), Error: bcd.ErrInvalidStringLength},
		{Input: []byte("17179869184:Sofia"), Error: bcd.ErrInvalidStringLength},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			_, err := bcd.DecodeString(reader)
			want := tc.Error
			assertError(t, err, want)
		})
	}
}

func TestDecodeInt(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output bcd.BInt
	}{
		{Input: []byte("i42e<whatever else is there>"), Output: 42},
		{Input: []byte("i244e<gibberish>"), Output: 244},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %d", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bcd.DecodeInt(reader)

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

		{Input: []byte("X23"), Error: bcd.ErrMalformedInput},

		{Input: []byte("ie"), Error: bcd.ErrInvalidIntegerValue},
		{Input: []byte("i e"), Error: bcd.ErrInvalidIntegerValue},
		{Input: []byte("i-+0e"), Error: bcd.ErrInvalidIntegerValue},
		{Input: []byte("i-0e"), Error: bcd.ErrInvalidIntegerValue},
		{Input: []byte("i-00e"), Error: bcd.ErrInvalidIntegerValue},
		{Input: []byte("i-00123e"), Error: bcd.ErrInvalidIntegerValue},
		{Input: []byte("i-123.32e"), Error: bcd.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			_, err := bcd.DecodeInt(reader)
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
		{[]byte("le"), bcd.BList{}},
		{[]byte("i42e"), bcd.BInt(42)},

		{[]byte("0:"), bcd.BString("")},
		{[]byte("3:Ana"), bcd.BString("Ana")},

		{[]byte("l3:anae"), bcd.BList{bcd.BString("ana")}},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %s", tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			got, err := bcd.DecodeAny(reader)
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
		{[]byte("X"), bcd.ErrMalformedInput},
		{[]byte("ie"), bcd.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		reader := newBufferedReader(tc.Input)

		_, err := bcd.DecodeAny(reader)
		want := tc.Err
		assertError(t, err, want)
	}
}

func TestDecodeList(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output bcd.BList
	}{
		{
			[]byte("le"),
			[]any{},
		},
		{
			[]byte("li32e3:Anae"),
			[]any{bcd.BInt(32), bcd.BString("Ana")},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %d", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bcd.DecodeList(reader)

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
		{[]byte("X"), bcd.ErrMalformedInput},

		{[]byte(""), io.ErrUnexpectedEOF},
		{[]byte("l"), io.ErrUnexpectedEOF},
		{[]byte("li12"), io.ErrUnexpectedEOF},
		{[]byte("li12e"), io.ErrUnexpectedEOF},

		{[]byte("li12e32:e"), io.ErrUnexpectedEOF},
		{[]byte("li12e:e"), bcd.ErrMalformedInput},
		{[]byte("li012ee"), bcd.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			_, err := bcd.DecodeList(reader)
			want := tc.Error
			assertError(t, err, want)
		})
	}
}

func TestDecodeDict(t *testing.T) {
	testCases := []struct {
		Input  []byte
		Output bcd.BDict
	}{
		{[]byte("de"), bcd.BDict{}},
		{
			[]byte("d3:ana5:sofia4:spaml1:a1:bee"),
			bcd.BDict{
				"ana":  bcd.BString("sofia"),
				"spam": bcd.BList{bcd.BString("a"), bcd.BString("b")},
			},
		},
		{
			Input: []byte("d4:infod6:lengthi1024e4:pathl4:dir14:dir28:file.txtee4:listl1:ad4:type4:teste1:bee"),
			Output: bcd.BDict{
				"info": bcd.BDict{
					"length": bcd.BInt(1024),
					"path": bcd.BList{
						bcd.BString("dir1"),
						bcd.BString("dir2"),
						bcd.BString("file.txt"),
					},
				},
				"list": bcd.BList{
					bcd.BString("a"),
					bcd.BDict{
						"type": bcd.BString("test"),
					},
					bcd.BString("b"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("decode %q to %v", tc.Input, tc.Output), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)

			want := tc.Output
			got, err := bcd.DecodeDict(reader)

			remainig := bytes.Buffer{}
			reader.WriteTo(&remainig)
			t.Logf("Remaining buffer: %q", &remainig)

			assertNoError(t, err)
			helpers.AssertEqual(t, want, got)
		})
	}
}

func TestDecodeDictErrors(t *testing.T) {
	testCases := []struct {
		Input []byte
		Error error
	}{
		{[]byte("X"), bencoding.ErrMalformedInput},

		{[]byte(""), io.ErrUnexpectedEOF},
		{[]byte("d"), io.ErrUnexpectedEOF},
		{[]byte("d4"), io.ErrUnexpectedEOF},
		{[]byte("d4:"), io.ErrUnexpectedEOF},
		{[]byte("d4:Sofia"), bencoding.ErrMalformedInput},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("returns %q when parsing %q", tc.Error, tc.Input), func(t *testing.T) {
			reader := newBufferedReader(tc.Input)
			_, err := bcd.DecodeDict(reader)
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
	t.Helper()
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
	t.Helper()
	if err != nil {
		t.Fatalf("an unexpected error was returned: %q", err)
	}
}
