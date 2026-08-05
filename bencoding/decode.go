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
	BString string
	BList   []any
	BDict   map[string]any
)

const StringMaxLength = 64 * 1024 * 1024

// Decode a bencoded string and returns it as an array of bytes.
// The function expects the string to be at the beginning of the stream
// and respect the BEP3 specifications. Any io.EOF erros encountered during
// parsing are treated as an io.UnexpectedEOF.
// If the length is invalid it returns Atoi-specific
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
	}
	if length < 0 {
		err = fmt.Errorf("%w: length cannot be negative", ErrInvalidStringLength)
		return
	}
	if length > StringMaxLength {
		err = fmt.Errorf("%w: length larger than the allowed limit of %d bytes", ErrInvalidStringLength, StringMaxLength)
		return
	}

	buffer := make([]byte, length)
	_, err = io.ReadFull(reader, buffer)
	data = BString(buffer)
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}

	return
}

// Parses an integer in the bencoding format. If the sequence
// doesn't start with an `i` the stream is considered malformed
// and an ErrMalformedInput is returned. If the value of the
// integered is invalid ErrInvalidIntegerValue will be returned,
// optionally alongside some Atoi-specific errors
func DecodeInt(reader *bufio.Reader) (value BInt, err error) {
	got, err := reader.ReadByte()
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}
	if got != 'i' {
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

// Parses a list in the bencoding format. If the sequence
// doesn't start with an `l` the stream is considered malformed
// and an ErrMalformedInput is returned. Any errors encountered
// while parsing the elements of the list are returned to the caller
func DecodeList(reader *bufio.Reader) (list BList, err error) {
	list = make([]any, 0)
	got, err := reader.ReadByte()
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}
	if got != 'l' {
		err = fmt.Errorf("%w: a bencoded list must start with byte `l`", ErrMalformedInput)
		return
	}

	for {
		yes, err := areWeFinished(reader)
		if err = helpers.TreatEOFAsUnexpected(err); err != nil {
			return nil, err
		}
		if yes {
			break
		}

		element, err := DecodeAny(reader)
		if err != nil {
			return nil, err
		}

		list = append(list, element)
	}

	// This will read the last 'e' (and we can
	// surely read it because previously we peeked
	// at it)
	reader.ReadByte()

	return
}

// Parses a dictionary in the bencoding format. If the sequence
// doesn't start with an `d` the stream is considered malformed
// and an ErrMalformedInput is returned. Any errors encountered
// while parsing the values and keys of the dictionary are returned
// to the caller
func DecodeDict(reader *bufio.Reader) (dict BDict, err error) {
	dict = make(map[string]any)

	got, err := reader.ReadByte()
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}
	if got != 'd' {
		err = fmt.Errorf("%w: a bencoded dictionary must start with byte `d`", ErrMalformedInput)
		return
	}

	for {
		yes, err := areWeFinished(reader)
		if err = helpers.TreatEOFAsUnexpected(err); err != nil {
			return nil, err
		}
		if yes {
			break
		}

		key, err := DecodeString(reader)
		if err != nil {
			return nil, err
		}

		value, err := DecodeAny(reader)
		if err != nil {
			return nil, err
		}

		dict[string(key)] = value
	}

	// no need to check for erros here
	reader.ReadByte()

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
	}
	if discriminator == 'l' {
		return DecodeList(reader)
	}
	if discriminator == 'd' {
		return DecodeDict(reader)
	}
	if isDigit(discriminator) {
		return DecodeString(reader)
	}

	err = fmt.Errorf("%w: discriminator %q doesn't correspond to any known type", ErrMalformedInput, discriminator)
	return nil, err
}

// Checks to see if the next byte is `e` (signaling the
// end of a list or a dictionary). It also forwards
// any io errors received
func areWeFinished(reader *bufio.Reader) (bool, error) {
	buffer, err := reader.Peek(1)
	if err != nil {
		return true, err
	}
	mark := buffer[0]
	return mark == 'e', err
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}
