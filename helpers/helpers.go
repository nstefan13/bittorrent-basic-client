package helpers

import (
	"errors"
	"io"
)

// A small adapter function that returns io.ErrUnexpectedEOF
// if err contains io.EOF
func TreatEOFAsUnexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
