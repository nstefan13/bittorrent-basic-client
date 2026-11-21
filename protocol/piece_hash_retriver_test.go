package protocol_test

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/protocol"
)

func TestPieceHashStore(t *testing.T) {
	t.Run("constructing a PieceHashStore panics if the string is empty", func(t *testing.T) {
		assert.Panics(t, func() {
			protocol.NewPieceHashStore("")
		})
	})

	t.Run("constructing a PieceHashStore panics if the string's length is not a multiple of 20", func(t *testing.T) {
		assert.Panics(t, func() {
			protocol.NewPieceHashStore("\x01\x02\x03")
		})
	})

	t.Run("accessing the hash of a non existing piece panics", func(t *testing.T) {
		assert.Panics(t, func() {
			testSubject := protocol.NewPieceHashStore("01234567890123456789")
			testSubject.GetHash(1)
		})
	})

	t.Run("accessing a negative piece index panics", func(t *testing.T) {
		assert.Panics(t, func() {
			testSubject := protocol.NewPieceHashStore("01234567890123456789")
			testSubject.GetHash(-1)
		})
	})

	t.Run("accessing an index equal to or greater than piece count panics", func(t *testing.T) {
		testSubject := protocol.NewPieceHashStore("01234567890123456789" + "12345678901234567890")
		assert.Panics(t, func() {
			testSubject.GetHash(2)
		})
		assert.Panics(t, func() {
			testSubject.GetHash(10)
		})
	})

	t.Run("retrieves the hash and total length from a valid piece hash string", func(t *testing.T) {
		p0 := "01234567890123456789"
		p1 := "12345678901234567890"
		p2 := "23456789012345678901"
		testSubject := protocol.NewPieceHashStore(p0 + p1 + p2)

		helpers.AssertEqual(t, 3, testSubject.Length())
		helpers.AssertEqual(t, [20]byte([]byte(p0)), testSubject.GetHash(0))
		helpers.AssertEqual(t, [20]byte([]byte(p1)), testSubject.GetHash(1))
		helpers.AssertEqual(t, [20]byte([]byte(p2)), testSubject.GetHash(2))
	})
}

// a small, untested, test utility
type ArrayBasedHashRetriver struct {
	hashes [][20]byte
}

func (s ArrayBasedHashRetriver) Length() int {
	return len(s.hashes)
}

func (s ArrayBasedHashRetriver) GetHash(index int) [20]byte {
	return s.hashes[index]
}
