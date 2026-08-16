package datastructures

import (
	"fmt"
	"strings"
)

// Bitfield represents a packed bit array used to track piece or block availability,
// where the bit at index 0 corresponds to the most significant bit (MSB) of the first byte.
type Bitfield struct {
	data []byte
}

// NewBitfield initializes a new [Bitfield] wrapping the provided byte slice.
// If the slice is empty or nil, it initializes a [Bitfield] with a single zero byte.
func NewBitfield(bytes []byte) *Bitfield {
	if len(bytes) == 0 {
		bytes = []byte{0}
	}

	return &Bitfield{bytes}
}

// NewBitfieldFromBits initializes a new [Bitfield] from a sequence of integers.
// A value of 1 sets the corresponding bit to 1, while any other value sets it to 0.
// Bits are packed high-bit first (MSB to LSB) within each byte.
// When called with no arguments, it initializes a [Bitfield] with a single zero byte.
func NewBitfieldFromBits(bits ...int) *Bitfield {
	if len(bits) == 0 {
		return &Bitfield{data: []byte{0}}
	}

	data := make([]byte, (len(bits)+7)/8)
	for i, value := range bits {
		if value == 1 {
			byteIndex := i / 8
			bitIndex := i & 0b111

			data[byteIndex] ^= (1 << (7 - bitIndex))
		}
	}

	return &Bitfield{data}
}

// SetBit sets (if value is true) or clears (if value is false) the bit at the specified index.
// If index requires more bytes than currently allocated, the underlying byte slice
// is dynamically expanded to accommodate it.
// Note: the bitfield is never shrunk, and trailing zero bytes are preserved.
//
// Example:
//
//	bf := NewBitfield(nil) // [00000000]
//	bf.SetBit(0, true)  // [10000000]
//	bf.SetBit(2, true)  // [10100000]
//	bf.SetBit(2, false) // [10000000]
//	bf.SetBit(11, true) // [10000000, 00010000]
func (b *Bitfield) SetBit(index uint, value bool) {
	// Step 1: increase the size of the data so we can address the bit at the specified index
	requiredSize := int(index/8 + 1)
	if requiredSize > len(b.data) {
		delta := requiredSize - len(b.data)
		for range delta {
			b.data = append(b.data, 0)
		}
	}

	// Step 2: set the value at the specified index
	byteIndex := index / 8
	bitIndex := index & 0b111
	if value {
		b.data[byteIndex] |= (1 << (7 - bitIndex))
	} else {
		b.data[byteIndex] &= ^(1 << (7 - bitIndex))
	}
}

// GetBit returns the boolean value of the bit at the specified index.
// If the index exceeds the currently allocated bytes, it returns false.
func (b Bitfield) GetBit(index uint) bool {
	byteIndex := int(index / 8)
	if byteIndex >= len(b.data) {
		return false
	}

	bitIndex := index & 0b111
	return (b.data[byteIndex] & (1 << (7 - bitIndex))) != 0
}

// ToBytes returns the underlying byte slice representing the bitfield.
func (b *Bitfield) ToBytes() []byte {
	return b.data
}

func (b *Bitfield) String() string {
	var sb strings.Builder

	for _, val := range b.data {
		sb.WriteString(fmt.Sprintf("%08b", val))
	}

	return sb.String()
}
