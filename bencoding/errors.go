package bencoding

import "errors"

var (
	ErrInvalidStringLength = errors.New("invalid length for bencoded string")
	ErrInvalidIntegerValue = errors.New("invalid value for bencoded integer")
	ErrMalformedInput      = errors.New("malformed input")
)

var ErrUnsupportedType = errors.New("type not supported for encoding")
