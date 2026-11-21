package protocol

import "fmt"

// PieceHashRetriver provides access to the expected SHA-1 hashes of torrent pieces and the total piece count.
type PieceHashRetriver interface {
	// GetHash returns the 20-byte SHA-1 hash for the piece at the specified index.
	GetHash(pieceIndex int) [20]byte

	// Length returns the total number of pieces.
	Length() int
}

// PieceHashStore is an in-memory implementation of [PieceHashRetriver] backed by a
// contiguous string of concatenated 20-byte SHA-1 hashes (e.g. metainfo.Pieces).
type PieceHashStore struct {
	hashes string
}

// NewPieceHashStore initializes a new [PieceHashStore] from a concatenated string of 20-byte SHA-1 piece hashes.
//
// Panics:
//   - If hashes is empty.
//   - If the length of hashes is not a multiple of 20.
func NewPieceHashStore(hashes string) PieceHashStore {
	if len(hashes) == 0 {
		panic("piece hashes string cannot be empty")
	}
	if len(hashes)%20 != 0 {
		panic(fmt.Sprintf("piece hashes string length (%d) must be a multiple of 20", len(hashes)))
	}

	return PieceHashStore{hashes: hashes}
}

// Length returns the total number of pieces stored in the [PieceHashStore].
func (s PieceHashStore) Length() int {
	return len(s.hashes) / 20
}

// GetHash returns the 20-byte SHA-1 hash for the piece at pieceIndex.
//
// Panics:
//   - If pieceIndex is negative.
//   - If pieceIndex is greater than or equal to [PieceHashStore.Length].
func (s PieceHashStore) GetHash(pieceIndex int) [20]byte {
	if pieceIndex < 0 || pieceIndex >= s.Length() {
		panic(fmt.Sprintf("piece index %d out of bounds (length %d)", pieceIndex, s.Length()))
	}

	start := pieceIndex * 20
	end := start + 20

	var hash [20]byte
	copy(hash[:], s.hashes[start:end])
	return hash
}
