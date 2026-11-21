package helpers

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"reflect"
	"strconv"
	"testing"

	"github.com/alecthomas/assert/v2"
)

// TreatEOFAsUnexpected translates [io.EOF] errors into [io.ErrUnexpectedEOF] to signal premature stream termination.
func TreatEOFAsUnexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}

// AssertEqual asserts that expected and got are equal, performing quote normalization when comparing strings.
func AssertEqual(t testing.TB, expected, got any) {
	t.Helper()

	typeOfExpected := reflect.TypeOf(expected).Kind()
	typeOfGot := reflect.TypeOf(got).Kind()
	typeOfString := reflect.TypeFor[string]().Kind()

	if typeOfString == typeOfExpected && typeOfExpected == typeOfGot {
		assert.Equal(t, fmt.Sprintf("%q", expected), fmt.Sprintf("%q", got))
	} else {
		assert.Equal(t, expected, got)
	}
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("an unexpected error was returned: %q", err)
	}
}

// AssertError fails the test if err is nil or does not match want via [errors.Is].
func AssertError(t testing.TB, err, want error) {
	t.Helper()
	if want == nil {
		panic("assertError called with `want` set to nil")
	}

	if err == nil {
		t.Fatal("expected an error to be returned")
	}

	if !errors.Is(err, want) {
		t.Errorf("expected %q, got %q", want, err.Error())
	}
}

// UrlFromString parses a raw URL string into a [url.URL], panicking if invalid.
func UrlFromString(str string) url.URL {
	result, err := url.Parse(str)
	if err != nil {
		panic(fmt.Sprintf("%q is not a valid URL", str))
	}
	return *result
}

// SHA1String computes the 20-byte SHA-1 hash of the concatenated input strings.
func SHA1String(strings ...string) [20]byte {
	var buffer []byte
	for _, str := range strings {
		buffer = append(buffer, []byte(str)...)
	}
	return sha1.Sum(buffer)
}

// Reads exactly `length` bytes from the provided `reader`. It forwards any
// IO errors it received back to the caller, with the exception of io.EOF, which
// always gets translated into a io.UnexpectedEOF (since this function is supposed
// to read exactly `length` bytes and not doing so should result in a error - like
// io.UnexpectedEOF, not a signal - like io.EOF)
func ReadExactly(length int, reader io.Reader) (buffer []byte, err error) {
	buffer = make([]byte, length)
	_, err = io.ReadFull(reader, buffer)
	err = TreatEOFAsUnexpected(err)
	return
}

func RandomHash() [20]byte {
	return SHA1String(strconv.Itoa(rand.Int()))
}
