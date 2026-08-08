package bencoding

import (
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
)

// Encoder serializes bencoding types (BInt, BString, BList, BDict) into an [io.Writer].
type Encoder struct {
	writer io.Writer
}

// NewEncoder creates a new bencoding [Encoder] writing to the provided writer.
func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{
		writer: writer,
	}
}

// EncodeString encodes a [BString] into bencoded format ("<length>:<content>").
func (e *Encoder) EncodeString(str BString) error {
	length := []byte(strconv.Itoa(len(str)))
	content := []byte(str)

	var data []byte
	data = length
	data = append(data, ':')
	data = append(data, content...)

	_, err := e.writer.Write(data)
	return err
}

// EncodeInt encodes a [BInt] into bencoded format ("i<integer>e").
func (e *Encoder) EncodeInt(num BInt) error {
	value := []byte(strconv.Itoa(int(num)))

	var data []byte
	data = []byte("i")
	data = append(data, value...)
	data = append(data, 'e')

	_, err := e.writer.Write(data)
	return err
}

// EncodeList encodes a [BList] into bencoded format ("l<elements>e").
// Elements are encoded sequentially using [Encoder.EncodeAny].
func (e *Encoder) EncodeList(list BList) error {
	_, err := e.writer.Write([]byte("l"))
	if err != nil {
		return err
	}

	for index, element := range list {
		err := e.EncodeAny(element)
		if err != nil {
			return fmt.Errorf("EncodeList: error while encoding the %d-th element: %w", index, err)
		}
	}

	_, err = e.writer.Write([]byte("e"))
	if err != nil {
		return err
	}

	return nil
}

// EncodeDict encodes a [BDict] into bencoded format ("d<key><val>...e").
//
// Per the BEP3 specification, dictionary keys must be sorted in lexicographical order (byte-by-byte comparison).
func (e *Encoder) EncodeDict(dict BDict) error {
	_, err := e.writer.Write([]byte("d"))
	if err != nil {
		return err
	}

	sortedKeys := slices.Collect(maps.Keys(dict))
	slices.Sort(sortedKeys)
	for _, key := range sortedKeys {
		value := dict[key]

		err = e.EncodeString(BString(key))
		if err != nil {
			return fmt.Errorf("EncodeDict: error while encoding a key: %w", err)
		}

		err = e.EncodeAny(value)
		if err != nil {
			return fmt.Errorf("EncodeDict: error while encoding the value for key %q: %w", key, err)
		}
	}

	_, err = e.writer.Write([]byte("e"))
	if err != nil {
		return err
	}

	return nil
}

// EncodeAny encodes a value by asserting its type against [BInt], [BString], [BList], and [BDict]
// and delegating to the appropriate Encode* method.
//
// Values must be typed as one of the explicit B* types. As such, passing primitive Go types (e.g. `int` instead of [BInt])
// will return [ErrUnsupportedType]. For example:
//
//	EncodeAny(52) returns a ErrUnsupportedType, while
//	EncodeAny(BInt(52)) succeeds
func (e *Encoder) EncodeAny(value any) error {
	if value, ok := value.(BInt); ok {
		return e.EncodeInt(value)
	}

	if value, ok := value.(BString); ok {
		return e.EncodeString(value)
	}

	if value, ok := value.(BList); ok {
		return e.EncodeList(value)
	}

	if value, ok := value.(BDict); ok {
		return e.EncodeDict(value)
	}

	actualType := reflect.TypeOf(value).Name()
	return fmt.Errorf("%w: tried to encode a value of type %s", ErrUnsupportedType, actualType)
}
