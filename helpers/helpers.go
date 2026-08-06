package helpers

import (
	"errors"
	"fmt"
	"io"
	"net/url"
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
		assert.Equal(t, fmt.Sprintf("%q", expected), fmt.Sprintf("%q", got))
	} else {
		assert.Equal(t, expected, got)
	}
}

func AssertNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("an unexpected error was returned: %q", err)
	}
}

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

func UrlFromString(str string) url.URL {
	result, err := url.Parse(str)
	if err != nil {
		panic(fmt.Sprintf("%q is not a valid URL", str))
	}
	return *result
}
