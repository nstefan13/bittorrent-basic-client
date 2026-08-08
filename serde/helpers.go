package serde

import (
	"bytes"
	"fmt"
	"reflect"

	bcd "github.com/nstefan13/bittorrent/bencoding"
)

// Decode decodes a bencoded byte stream into the specified target type T (such as [bcd.BString], [bcd.BInt], [bcd.BList], or [bcd.BDict]).
// Returns [ErrTypeMismatch] if the decoded value cannot be asserted to T.
func Decode[T any](stream []byte) (T, error) {
	var zero T
	decoder := bcd.NewDecoder(bytes.NewReader(stream))

	untypedValue, err := decoder.DecodeAny()
	if err != nil {
		return zero, err
	}

	typedValue, ok := untypedValue.(T)
	if !ok {
		return zero, fmt.Errorf("%w: value should be of type %T, not %T", ErrTypeMismatch, zero, untypedValue)
	}

	return typedValue, nil
}

// Encode serializes a bencoding value (such as [bcd.BString], [bcd.BInt], [bcd.BList], or [bcd.BDict]) into a bencoded byte slice.
func Encode(data any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	err := bcd.NewEncoder(buffer).EncodeAny(data)
	return buffer.Bytes(), err
}


// ExtractValueFromBDict retrieves the value associated with key from the bencode dictionary dict
// and safely asserts its type to ValueType (such as BString, BInt, BDict, or BList).
//
// Returns:
//   - KeyNotFoundError (unwrapping to ErrKeyNotFound) if key is missing from dict.
//   - An error wrapping ErrTypeMismatch if key exists but holds a value of a different bencode type,
//     then the one specified in `ValueType`
func ExtractValueFromBDict[ValueType any](dict bcd.BDict, key string) (value ValueType, err error) {
	untypedValue, alreadyThere := dict[key]
	if !alreadyThere {
		err = &KeyNotFoundError{key}
		return
	}

	value, ok := dict[key].(ValueType)
	if !ok {
		err = fmt.Errorf("%w: value corresponding to key %q should be of type %q, not %q", ErrTypeMismatch, key, reflect.TypeFor[ValueType]().Name(), reflect.TypeOf(untypedValue).Name())
		return
	}

	return
}
