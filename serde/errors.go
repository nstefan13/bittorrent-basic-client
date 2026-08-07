package serde

import (
	"errors"
	"fmt"
)

// Sentinel Errors
var (
	// pacakge wide
	ErrKeyNotFound  = errors.New("key does not exists")
	ErrTypeMismatch = errors.New("value has incorrect type")

	// specific for metainfo.go
	ErrMalformedMetaInfo = errors.New("torrent is corrupt")
)

type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %q not found", e.Key)
}

func (e *KeyNotFoundError) Unwrap() error {
	return ErrKeyNotFound
}

type MalformedMetaInfoError struct {
	Err error
}

func (e *MalformedMetaInfoError) Error() string {
	return fmt.Sprintf("%v: %v", ErrMalformedMetaInfo, e.Err)
}

func (e *MalformedMetaInfoError) Unwrap() []error {
	return []error{ErrMalformedMetaInfo, e.Err}
}
