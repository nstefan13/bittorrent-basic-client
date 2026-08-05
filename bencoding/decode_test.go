package bencoding_test

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
)

func TestDecodeString(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  []byte
		Output bcd.BString
	}{
		{
			Name:   "valid string with trailing content",
			Input:  []byte("5:SofiaAAA"),
			Output: bcd.BString("Sofia"),
		},
		{
			Name:   "short string with extra trailing content",
			Input:  []byte("3:AnaASASlkDJLJASK"),
			Output: bcd.BString("Ana"),
		},
		{
			Name:   "binary string with null bytes",
			Input:  []byte("3:\x00\x01\x03"),
			Output: bcd.BString("\x00\x01\x03"),
		},
		{
			Name:   "empty string",
			Input:  []byte("0:"),
			Output: bcd.BString(""),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			got, err := decoder.DecodeString()
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, got)
		})
	}
}

func TestDecodeStringErrors(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
		Error error
	}{
		{Name: "empty input", Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Name: "missing colon and string data", Input: []byte("5"), Error: io.ErrUnexpectedEOF},
		{Name: "missing length prefix and colon", Input: []byte("Sofia"), Error: io.ErrUnexpectedEOF},
		{Name: "missing string data after colon", Input: []byte("5:"), Error: io.ErrUnexpectedEOF},
		{Name: "string data shorter than specified length", Input: []byte("10000:too_small_of_a_string"), Error: io.ErrUnexpectedEOF},
		{Name: "empty length prefix before colon", Input: []byte(":"), Error: bcd.ErrInvalidStringLength},
		{Name: "negative length prefix", Input: []byte("-1:"), Error: bcd.ErrInvalidStringLength},
		{Name: "leading space in length prefix", Input: []byte(" 5:Sofia"), Error: bcd.ErrInvalidStringLength},
		{Name: "space before colon in length prefix", Input: []byte("5 :Sofia"), Error: bcd.ErrInvalidStringLength},
		{Name: "length prefix is greater than the max allowed length", Input: []byte("17179869184:Sofia"), Error: bcd.ErrInvalidStringLength},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			_, err := decoder.DecodeString()
			helpers.AssertError(t, err, tc.Error)
		})
	}
}

func TestDecodeInt(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  []byte
		Output bcd.BInt
	}{
		{
			Name:   "positive integer with trailing data",
			Input:  []byte("i42e<whatever else is there>"),
			Output: 42,
		},
		{
			Name:   "positive integer with trailing gibberish",
			Input:  []byte("i244e<gibberish>"),
			Output: 244,
		},
		{
			Name:   "zero",
			Input:  []byte("i0e"),
			Output: 0,
		},
		{
			Name:   "negative integer",
			Input:  []byte("i-234e"),
			Output: -234,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			got, err := decoder.DecodeInt()
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, got)
		})
	}
}

func TestDecodeIntErrors(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
		Error error
	}{
		{Name: "empty input", Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Name: "only prefix 'i'", Input: []byte("i"), Error: io.ErrUnexpectedEOF},
		{Name: "positive integer missing trailing 'e'", Input: []byte("i23"), Error: io.ErrUnexpectedEOF},
		{Name: "negative integer missing trailing 'e'", Input: []byte("i-123"), Error: io.ErrUnexpectedEOF},
		{Name: "missing 'i' prefix", Input: []byte("X23"), Error: bcd.ErrMalformedInput},
		{Name: "empty value between 'i' and 'e'", Input: []byte("ie"), Error: bcd.ErrInvalidIntegerValue},
		{Name: "value containing space", Input: []byte("i e"), Error: bcd.ErrInvalidIntegerValue},
		{Name: "invalid sign prefix '-+'", Input: []byte("i-+0e"), Error: bcd.ErrInvalidIntegerValue},
		{Name: "negative zero '-0'", Input: []byte("i-0e"), Error: bcd.ErrInvalidIntegerValue},
		{Name: "negative zero with leading zeros '-00'", Input: []byte("i-00e"), Error: bcd.ErrInvalidIntegerValue},
		{Name: "leading zero in negative integer '-00123'", Input: []byte("i-00123e"), Error: bcd.ErrInvalidIntegerValue},
		{Name: "floating point number", Input: []byte("i-123.32e"), Error: bcd.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			_, err := decoder.DecodeInt()
			helpers.AssertError(t, err, tc.Error)
		})
	}
}

func TestDecodeAny(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  []byte
		Output any
	}{
		{Name: "decode empty list", Input: []byte("le"), Output: bcd.BList{}},
		{Name: "decode integer", Input: []byte("i42e"), Output: bcd.BInt(42)},
		{Name: "decode empty string", Input: []byte("0:"), Output: bcd.BString("")},
		{Name: "decode string", Input: []byte("3:Ana"), Output: bcd.BString("Ana")},
		{Name: "decode list with string", Input: []byte("l3:anae"), Output: bcd.BList{bcd.BString("ana")}},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			got, err := decoder.DecodeAny()
			helpers.AssertNoError(t, err)

			actualTypeName := typeNameOf(got)
			expectedTypeName := typeNameOf(tc.Output)
			if actualTypeName != expectedTypeName {
				t.Fatalf("expected return value to be of type %q, instead it is of type %q", expectedTypeName, actualTypeName)
			}

			helpers.AssertEqual(t, tc.Output, got)
		})
	}
}

func TestDecodeAnyErrors(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
		Error error
	}{
		{Name: "empty input", Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Name: "unknown discriminator byte", Input: []byte("X"), Error: bcd.ErrMalformedInput},
		{Name: "invalid integer element", Input: []byte("ie"), Error: bcd.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			_, err := decoder.DecodeAny()
			helpers.AssertError(t, err, tc.Error)
		})
	}
}

func TestDecodeList(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  []byte
		Output bcd.BList
	}{
		{
			Name:   "empty list",
			Input:  []byte("le"),
			Output: bcd.BList{},
		},
		{
			Name:   "list with integer and string",
			Input:  []byte("li32e3:Anae"),
			Output: bcd.BList{bcd.BInt(32), bcd.BString("Ana")},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			got, err := decoder.DecodeList()
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, got)
		})
	}
}

func TestDecodeListErrors(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
		Error error
	}{
		{Name: "missing 'l' prefix", Input: []byte("X"), Error: bcd.ErrMalformedInput},
		{Name: "empty input", Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Name: "only prefix 'l'", Input: []byte("l"), Error: io.ErrUnexpectedEOF},
		{Name: "unfinished integer element inside list", Input: []byte("li12"), Error: io.ErrUnexpectedEOF},
		{Name: "missing list terminator 'e'", Input: []byte("li12e"), Error: io.ErrUnexpectedEOF},
		{Name: "unfinished string element inside list", Input: []byte("li12e32:e"), Error: io.ErrUnexpectedEOF},
		{Name: "invalid length prefix for string inside list", Input: []byte("li12e:e"), Error: bcd.ErrMalformedInput},
		{Name: "invalid integer with leading zero inside list", Input: []byte("li012ee"), Error: bcd.ErrInvalidIntegerValue},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			_, err := decoder.DecodeList()
			helpers.AssertError(t, err, tc.Error)
		})
	}
}

func TestDecodeDict(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  []byte
		Output bcd.BDict
	}{
		{
			Name:   "empty dict",
			Input:  []byte("de"),
			Output: bcd.BDict{},
		},
		{
			Name:  "dict with string and list values",
			Input: []byte("d3:ana5:sofia4:spaml1:a1:bee"),
			Output: bcd.BDict{
				"ana":  bcd.BString("sofia"),
				"spam": bcd.BList{bcd.BString("a"), bcd.BString("b")},
			},
		},
		{
			Name:  "nested dict with list",
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
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			got, err := decoder.DecodeDict()
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, got)
		})
	}
}

func TestDecodeDictErrors(t *testing.T) {
	testCases := []struct {
		Name  string
		Input []byte
		Error error
	}{
		{Name: "missing 'd' prefix", Input: []byte("X"), Error: bcd.ErrMalformedInput},
		{Name: "empty input", Input: []byte(""), Error: io.ErrUnexpectedEOF},
		{Name: "only prefix 'd'", Input: []byte("d"), Error: io.ErrUnexpectedEOF},
		{Name: "incomplete dict key length prefix", Input: []byte("d4"), Error: io.ErrUnexpectedEOF},
		{Name: "incomplete dict key prefix", Input: []byte("d4:"), Error: io.ErrUnexpectedEOF},
		{Name: "dict key missing value", Input: []byte("d4:Sofia"), Error: bcd.ErrMalformedInput},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			decoder := bcd.NewDecoder(bytes.NewReader(tc.Input))
			_, err := decoder.DecodeDict()
			helpers.AssertError(t, err, tc.Error)
		})
	}
}

func typeNameOf(value any) string {
	return reflect.TypeOf(value).Name()
}
