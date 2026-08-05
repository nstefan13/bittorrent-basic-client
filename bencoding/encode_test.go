package bencoding_test

import (
	"bytes"
	"errors"
	"testing"

	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
)

type errWriter struct {
	err error
}

func (w *errWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

func TestEncodeString(t *testing.T) {
	testCases := []struct {
		Input  bcd.BString
		Output []byte
	}{
		{bcd.BString("\x00\x01\x03"), []byte("3:\x00\x01\x03")},
		{bcd.BString("Sofia"), []byte("5:Sofia")},
		{bcd.BString(""), []byte("0:")},
	}

	for _, tc := range testCases {
		buffer := &bytes.Buffer{}
		err := bcd.NewEncoder(buffer).EncodeString(tc.Input)
		helpers.AssertNoError(t, err)
		helpers.AssertEqual(t, tc.Output, buffer.Bytes())
	}
}

func TestEncodeInt(t *testing.T) {
	testCases := []struct {
		Input  bcd.BInt
		Output []byte
	}{
		{bcd.BInt(0), []byte("i0e")},
		{bcd.BInt(-0), []byte("i0e")},
		{bcd.BInt(32), []byte("i32e")},
		{bcd.BInt(-234), []byte("i-234e")},
	}

	for _, tc := range testCases {
		buffer := &bytes.Buffer{}
		err := bcd.NewEncoder(buffer).EncodeInt(tc.Input)
		helpers.AssertNoError(t, err)
		helpers.AssertEqual(t, tc.Output, buffer.Bytes())
	}
}

func TestEncodeList(t *testing.T) {
	testCases := []struct {
		Input  bcd.BList
		Output []byte
	}{
		{bcd.BList{bcd.BInt(2015), bcd.BString("Sofia")}, []byte("li2015e5:Sofiae")},
		{bcd.BList{}, []byte("le")},
	}

	for _, tc := range testCases {
		buffer := &bytes.Buffer{}
		err := bcd.NewEncoder(buffer).EncodeList(tc.Input)
		helpers.AssertNoError(t, err)
		helpers.AssertEqual(t, tc.Output, buffer.Bytes())
	}
}

func TestEncodeListErrors(t *testing.T) {
	t.Run("return ErrInvalidType if the list contains unsupported types", func(t *testing.T) {
		input := bcd.BList{bcd.BString("Ana"), bcd.BInt(12), 89}
		err := bcd.NewEncoder(&bytes.Buffer{}).EncodeList(input)
		helpers.AssertError(t, err, bcd.ErrUnsupportedType)
	})
}

func TestEncodeDict(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  bcd.BDict
		Output []byte
	}{
		{
			Name:   "empty dict",
			Input:  bcd.BDict{},
			Output: []byte("de"),
		},
		{
			Name: "dict with list value",
			Input: bcd.BDict{
				"spam": bcd.BList{bcd.BString("a"), bcd.BString("b")},
			},
			Output: []byte("d4:spaml1:a1:bee"),
		},
		{
			Name: "dict with string and list values",
			Input: bcd.BDict{
				"ana":  bcd.BString("sofia"),
				"spam": bcd.BList{bcd.BString("a"), bcd.BString("b")},
			},
			Output: []byte("d3:ana5:sofia4:spaml1:a1:bee"),
		},
		{
			Name: "nested dict",
			Input: bcd.BDict{
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
			Output: []byte("d4:infod6:lengthi1024e4:pathl4:dir14:dir28:file.txtee4:listl1:ad4:type4:teste1:bee"),
		},
		{
			Name: "the keys should be sorted",
			Input: bcd.BDict{
				"c":   bcd.BInt(1),
				"a":   bcd.BInt(1),
				"zz":  bcd.BInt(1),
				"bdf": bcd.BInt(1),
				"abc": bcd.BInt(1),
				"be":  bcd.BInt(1),
			},
			Output: []byte("d1:ai1e3:abci1e3:bdfi1e2:bei1e1:ci1e2:zzi1ee"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := bcd.NewEncoder(buffer).EncodeDict(tc.Input)
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, buffer.Bytes())
		})
	}
}

func TestEncodeDictErrors(t *testing.T) {
	t.Run("return ErrUnsupportedType for unsupported value types in dictionary", func(t *testing.T) {
		input := bcd.BDict{
			"key": 54,
		}
		err := bcd.NewEncoder(&bytes.Buffer{}).EncodeDict(input)
		helpers.AssertError(t, err, bcd.ErrUnsupportedType)
	})

	t.Run("return error when writer fails", func(t *testing.T) {
		writeErr := errors.New("write failure")
		w := &errWriter{err: writeErr}
		err := bcd.NewEncoder(w).EncodeDict(bcd.BDict{})
		helpers.AssertError(t, err, writeErr)
	})
}

func TestEncodeAny(t *testing.T) {
	testCases := []struct {
		Input  any
		Output []byte
	}{
		{bcd.BInt(54), []byte("i54e")},
		{bcd.BString("AnaSofia"), []byte("8:AnaSofia")},
		{bcd.BString(""), []byte("0:")},
	}

	for _, tc := range testCases {
		buffer := &bytes.Buffer{}
		err := bcd.NewEncoder(buffer).EncodeAny(tc.Input)
		helpers.AssertNoError(t, err)
		helpers.AssertEqual(t, tc.Output, buffer.Bytes())
	}
}

func TestEncodeAnyErrors(t *testing.T) {
	t.Run("return ErrInvalidType for types not supported by the encoder", func(t *testing.T) {
		input := 54
		err := bcd.NewEncoder(&bytes.Buffer{}).EncodeAny(input)
		helpers.AssertError(t, err, bcd.ErrUnsupportedType)
	})
}
