package bencoding

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/nstefan13/bittorrent/helpers"
)

var ErrInvalidStringLength = errors.New("invalid length for bencoded string")

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
func DecodeString(reader *bufio.Reader) (data []byte, err error) {
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
