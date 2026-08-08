package bencoding

import "io"

// TrackedReader wraps an [io.Reader] to keep track of the total number of bytes read (offset).
type TrackedReader struct {
	reader io.Reader
	offset int
}

// Read reads up to len(p) bytes from the underlying reader and updates the byte offset count.
func (tr *TrackedReader) Read(p []byte) (n int, err error) {
	n, err = tr.reader.Read(p)
	tr.offset += n
	return
}

