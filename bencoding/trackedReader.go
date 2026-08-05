package bencoding

import "io"

// An struct wrapping a custom reader in order to track
// the the offset (aka. how many bytes have been read)
type TrackedReader struct {
	reader io.Reader
	offset int
}

func (tr *TrackedReader) Read(p []byte) (n int, err error) {
	n, err = tr.reader.Read(p)
	tr.offset += n
	return
}
