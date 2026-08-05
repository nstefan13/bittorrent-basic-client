package bencoding

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/nstefan13/bittorrent/helpers"
)

var (
	ErrInvalidStringLength = errors.New("invalid length for bencoded string")
	ErrInvalidIntegerValue = errors.New("invalid value for bencoded integer")
	ErrMalformedInput      = errors.New("malformed input")
)

type (
	BInt    int
	BString []byte
	BList   []any
	BDict   map[string]any
)

const StringMaxLength = 64 * 1024 * 1024

// Decode a bencoded string and returns it as an array of bytes.
// The function expects the string to be at the beginning of the stream
// and respect the BEP3 specifications. Any io.EOF erros encountered during
// parsing are treated as an io.UnexpectedEOF.
// If the length is invalid, Returns Atoi-specific
// errors (like strconv.ErrSyntax or strconv.ErrRange) wrapped alongside
// ErrInvalidStringLength.
// To prevent attackers from allocating huge chunks of memory for strings,
// the functions caps the length at 64MB (arbitrary number, not configurable)
func DecodeString(reader *bufio.Reader) (data BString, err error) {
	prefix, err := reader.ReadString(':')
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}

	prefix = prefix[:len(prefix)-1]
	length, err := strconv.Atoi(prefix)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrInvalidStringLength, err)
		return
	} else if length < 0 {
		err = fmt.Errorf("%w: length cannot be negative", ErrInvalidStringLength)
		return
	} else if length > StringMaxLength {
		err = fmt.Errorf("%w: length larger than the allowed limit of %d bytes", ErrInvalidStringLength, StringMaxLength)
		return
	}

	data = make([]byte, length)
	_, err = io.ReadFull(reader, data)
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}

	return
}

func DecodeInt(reader *bufio.Reader) (value BInt, err error) {
	got, err := reader.ReadByte()
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	} else if got != 'i' {
		err = fmt.Errorf("%w: a bencoded integer must start with byte `i`", ErrMalformedInput)
		return
	}

	data, err := reader.ReadString('e')
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}

	data = data[:len(data)-1]
	valueAsInt, err := strconv.Atoi(data)
	if err != nil {
		err = fmt.Errorf("%w: %w", ErrInvalidIntegerValue, err)
		return
	}

	// by now, we are sure that the raw value was a valid integer
	// due to the use of Atoi, but there is still the possibility
	// of the input to contain leading zeros or the negative zero
	// (-0 or -00). Fortunately, this is quite easy to verify
	valueAsString := strconv.Itoa(valueAsInt)
	if valueAsString != data {
		err = fmt.Errorf("%w (data=%q)", ErrInvalidIntegerValue, data)
		return
	}

	value = BInt(valueAsInt)
	return
}

// Attempts to decode the beggining of the stream into one of
// the 4 types provided by the bencoding specifications. The
// underlying object it returns is:
//   - of type `BInt` if it decoded an bencoded integer
//   - of type `BString` if it decoded an bencoded string
//   - of type `BList` if it decoded an bencoded list
//   - of type `BDict` if it decoded an bencoded dictionary
//
// This function acts as a router for the other Decode* functions.
// It peeks at the first byte and uses it to guess what data
// might be encoded there. If it cannot deduce the type of data, it
// returns an ErrMalformedInput. If it successfully deduces the type
// of data, it calls the corresponding Decode* function and forwards
// the return values (including any errors)
func DecodeAny(reader *bufio.Reader) (any, error) {
	buffer, err := reader.Peek(1)
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return nil, err
	}

	discriminator := buffer[0]

	if discriminator == 'i' {
		return DecodeInt(reader)
	} else if isDigit(discriminator) {
		return DecodeString(reader)
	} else {
		err := fmt.Errorf("%w: discriminator %q doesn't correspond to any known type", ErrMalformedInput, discriminator)
		return nil, err
	}
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}
