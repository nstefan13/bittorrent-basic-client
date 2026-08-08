package protocol

import "errors"

// ErrMalformedTrackerResponse is returned when the tracker response violates the BEP3 specification
// (e.g. invalid bencode, missing peers key, or malformed peer list representations).
var ErrMalformedTrackerResponse = errors.New("tracker returned invalid response")

// ErrTrackerReportedFailure is returned when the tracker explicitly responds with a failure message via "failure reason".
var ErrTrackerReportedFailure = errors.New("tracker reported failure")

