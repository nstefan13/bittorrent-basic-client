package helpers

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/alecthomas/assert/v2"
)

// A small adapter function that returns io.ErrUnexpectedEOF
// if err contains io.EOF
func TreatEOFAsUnexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

func AssertEqual(t testing.TB, expected, got any) {
	t.Helper()

	typeOfExpected := reflect.TypeOf(expected).Kind()
	typeOfGot := reflect.TypeOf(got).Kind()
	typeOfString := reflect.TypeFor[string]().Kind()

	if typeOfString == typeOfExpected && typeOfExpected == typeOfGot {
		assert.Equal(t, fmt.Sprintf("%q", expected), fmt.Sprintf("%q", expected))
	} else {
		assert.Equal(t, expected, got)
	}
}
