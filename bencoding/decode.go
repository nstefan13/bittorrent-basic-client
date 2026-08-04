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

// Decode a bencoded string and returns it as an array of bytes.
// The function expects the string to be at the beginning of the stream
// and respect the BEP3 specifications. Any io.EOF erros encountered during
// parsing are treated as an io.UnexpectedEOF.
// Returns Atoi-specific errors (like strconv.ErrSyntax or strconv.ErrRange) if
// the length is invalid.
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
	}

	data = make([]byte, length)
	_, err = io.ReadFull(reader, data)
	if err = helpers.TreatEOFAsUnexpected(err); err != nil {
		return
	}

	return
}
