package bencoding

import "errors"

var (
	// ErrInvalidStringLength is returned when a bencoded string has an invalid length prefix
	// (e.g. non-numeric, negative, or exceeding BStringMaxLength).
	ErrInvalidStringLength = errors.New("invalid length for bencoded string")

	// ErrInvalidIntegerValue is returned when a bencoded integer is malformed
	// (e.g. empty, invalid characters, negative zero "-0", or contains leading zeros like "012").
	ErrInvalidIntegerValue = errors.New("invalid value for bencoded integer")

	// ErrMalformedInput is returned when an input stream does not match the expected bencoding syntax
	// or encounters an unknown type discriminator.
	ErrMalformedInput = errors.New("malformed input")
)

// ErrUnsupportedType is returned by the Encoder when encountering a value type that does not
// correspond to one of the supported bencoding types (BInt, BString, BList, BDict).
var ErrUnsupportedType = errors.New("type not supported for encoding")

