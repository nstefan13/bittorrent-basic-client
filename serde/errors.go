package serde

import (
	"errors"
	"fmt"
)

var (
	// ErrKeyNotFound is returned when an expected key is missing from a bencoded dictionary.
	ErrKeyNotFound = errors.New("key does not exists")

	// ErrTypeMismatch is returned when a dictionary entry exists but holds a different bencode type than expected.
	ErrTypeMismatch = errors.New("value has incorrect type")

	// ErrMalformedMetaInfo is returned when a .torrent metainfo dictionary violates the BEP3 specification.
	ErrMalformedMetaInfo = errors.New("torrent is corrupt")
)

// KeyNotFoundError represents an error where a specific key is missing from a bencoded dictionary.
// It unwraps to [ErrKeyNotFound].
type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %q not found", e.Key)
}

func (e *KeyNotFoundError) Unwrap() error {
	return ErrKeyNotFound
}

// MalformedMetaInfoError wraps an underlying metainfo validation error.
// It implements multi-error unwrapping to allow callers to match against both [ErrMalformedMetaInfo]
// and the underlying cause (such as [ErrKeyNotFound] or [ErrTypeMismatch]).
type MalformedMetaInfoError struct {
	Err error
}

func (e *MalformedMetaInfoError) Error() string {
	return fmt.Sprintf("%v: %v", ErrMalformedMetaInfo, e.Err)
}

func (e *MalformedMetaInfoError) Unwrap() []error {
	return []error{ErrMalformedMetaInfo, e.Err}
}

