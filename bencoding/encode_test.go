package bencoding_test

import (
	"bytes"
	"errors"
	"io"
	"slices"
	"testing"

	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
)

type errWriter struct {
	errGenerator func(*errWriter) error
	calls        [][]byte
}

func (w *errWriter) Write(p []byte) (n int, err error) {
	w.calls = append(w.calls, p)
	return len(p), w.errGenerator(w)
}

func TestEncodeString(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  bcd.BString
		Output []byte
	}{
		{
			Name:   "binary string",
			Input:  bcd.BString("\x00\x01\x03"),
			Output: []byte("3:\x00\x01\x03"),
		},
		{
			Name:   "regular string",
			Input:  bcd.BString("Sofia"),
			Output: []byte("5:Sofia"),
		},
		{
			Name:   "empty string",
			Input:  bcd.BString(""),
			Output: []byte("0:"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := bcd.NewEncoder(buffer).EncodeString(tc.Input)
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, buffer.Bytes())
		})
	}
}

func TestEncodeInt(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  bcd.BInt
		Output []byte
	}{
		{
			Name:   "zero",
			Input:  bcd.BInt(0),
			Output: []byte("i0e"),
		},
		{
			Name:   "negative zero",
			Input:  bcd.BInt(-0),
			Output: []byte("i0e"),
		},
		{
			Name:   "positive int",
			Input:  bcd.BInt(32),
			Output: []byte("i32e"),
		},
		{
			Name:   "negative int",
			Input:  bcd.BInt(-234),
			Output: []byte("i-234e"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := bcd.NewEncoder(buffer).EncodeInt(tc.Input)
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, buffer.Bytes())
		})
	}
}

func TestEncodeList(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  bcd.BList
		Output []byte
	}{
		{
			Name:   "list with int and string",
			Input:  bcd.BList{bcd.BInt(2015), bcd.BString("Sofia")},
			Output: []byte("li2015e5:Sofiae"),
		},
		{
			Name:   "empty list",
			Input:  bcd.BList{},
			Output: []byte("le"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := bcd.NewEncoder(buffer).EncodeList(tc.Input)
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, buffer.Bytes())
		})
	}
}

func TestEncodeListErrors(t *testing.T) {
	t.Run("return ErrUnsupportedType if the list contains unsupported types", func(t *testing.T) {
		input := bcd.BList{bcd.BString("Ana"), bcd.BInt(12), 89}
		err := bcd.NewEncoder(&bytes.Buffer{}).EncodeList(input)
		helpers.AssertError(t, err, bcd.ErrUnsupportedType)
	})

	t.Run("return error when writer fails", func(t *testing.T) {
		w := &errWriter{errGenerator: func(ew *errWriter) error { return io.ErrClosedPipe }}
		err := bcd.NewEncoder(w).EncodeList(bcd.BList{})
		helpers.AssertError(t, err, io.ErrClosedPipe)
	})

	t.Run("return error when writer fails at the end of list", func(t *testing.T) {
		w := &errWriter{errGenerator: func(ew *errWriter) error {
			if slices.Equal(ew.calls[len(ew.calls)-1], []byte("e")) {
				return io.ErrClosedPipe
			}
			return nil
		}}
		err := bcd.NewEncoder(w).EncodeList(bcd.BList{})
		helpers.AssertError(t, err, io.ErrClosedPipe)
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
		w := &errWriter{errGenerator: func(ew *errWriter) error { return writeErr }}
		err := bcd.NewEncoder(w).EncodeDict(bcd.BDict{})
		helpers.AssertError(t, err, writeErr)
	})

	t.Run("return error when writer fails at the end of dictionary", func(t *testing.T) {
		w := &errWriter{errGenerator: func(ew *errWriter) error {
			if slices.Equal(ew.calls[len(ew.calls)-1], []byte("e")) {
				return io.ErrClosedPipe
			}
			return nil
		}}
		err := bcd.NewEncoder(w).EncodeDict(bcd.BDict{})
		helpers.AssertError(t, err, io.ErrClosedPipe)
	})

	t.Run("return error when writer fails while encoding a key", func(t *testing.T) {
		w := &errWriter{errGenerator: func(ew *errWriter) error {
			if len(ew.calls) == 2 {
				return io.ErrClosedPipe
			}
			return nil
		}}
		err := bcd.NewEncoder(w).EncodeDict(bcd.BDict{"key": bcd.BInt(1)})
		helpers.AssertError(t, err, io.ErrClosedPipe)
	})
}

func TestEncodeAny(t *testing.T) {
	testCases := []struct {
		Name   string
		Input  any
		Output []byte
	}{
		{
			Name:   "encode int",
			Input:  bcd.BInt(54),
			Output: []byte("i54e"),
		},
		{
			Name:   "encode string",
			Input:  bcd.BString("AnaSofia"),
			Output: []byte("8:AnaSofia"),
		},
		{
			Name:   "encode empty string",
			Input:  bcd.BString(""),
			Output: []byte("0:"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			buffer := &bytes.Buffer{}
			err := bcd.NewEncoder(buffer).EncodeAny(tc.Input)
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tc.Output, buffer.Bytes())
		})
	}
}

func TestEncodeAnyErrors(t *testing.T) {
	t.Run("return ErrUnsupportedType for types not supported by the encoder", func(t *testing.T) {
		input := 54
		err := bcd.NewEncoder(&bytes.Buffer{}).EncodeAny(input)
		helpers.AssertError(t, err, bcd.ErrUnsupportedType)
	})
}
