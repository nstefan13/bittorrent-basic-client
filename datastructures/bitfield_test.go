package datastructures_test

import (
	"fmt"
	"slices"
	"testing"

	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/helpers"
)

func TestNewBitfield(t *testing.T) {
	t.Run("wraps bytes in a Bitfield", func(t *testing.T) {
		buffer := []byte{0b00010101, 0b01010010}
		bf := ds.NewBitfield(buffer)

		want := "0001010101010010"
		got := bf.String()
		helpers.AssertEqual(t, want, got)
	})

	t.Run("nil or empty byte slice is initialized with a single zero byte", func(t *testing.T) {
		bfNil := ds.NewBitfield(nil)
		assertBytes(t, []byte{0b00000000}, bfNil.ToBytes())

		bfEmpty := ds.NewBitfield([]byte{})
		assertBytes(t, []byte{0b00000000}, bfEmpty.ToBytes())
	})
}

func TestNewBitfieldFromBits(t *testing.T) {
	testCases := []struct {
		name  string
		input []int
		want  []byte
	}{
		{
			name:  "initialize bitfield from list of 10 integers",
			input: []int{0, 1, 1, 0, 1, 1, 1, 0, 0, 1},
			want:  []byte{0b01101110, 0b01000000},
		},
		{
			name:  "initialize bitfield from list of 9 integers",
			input: []int{0, 1, 1, 0, 1, 1, 1, 0, 1},
			want:  []byte{0b01101110, 0b10000000},
		},
		{
			name:  "initialize bitfield from list of 5 integers",
			input: []int{0, 1, 1, 0, 1},
			want:  []byte{0b01101000},
		},
		{
			name:  "initialize bitfield from list of 1 integer",
			input: []int{1},
			want:  []byte{0b10000000},
		},
		{
			name:  "any value different from 1 becomes a 0-bit",
			input: []int{1, -1, 2, 1},
			want:  []byte{0b10010000},
		},
		{
			name:  "an uninitialized is just one byte with all eight bits set to 0",
			input: nil,
			want:  []byte{0b00000000},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ds.NewBitfieldFromBits(tc.input...).ToBytes()
			assertBytes(t, tc.want, got)
		})
	}

	t.Run("dynamically grow bitfield and set/clear individual bits", func(t *testing.T) {
		bf := ds.NewBitfield(nil)
		assertBytes(t, []byte{0b00000000}, bf.ToBytes())

		bf.SetBit(0, true)
		assertBytes(t, []byte{0b10000000}, bf.ToBytes())

		bf.SetBit(1, true)
		assertBytes(t, []byte{0b11000000}, bf.ToBytes())

		bf.SetBit(1, false)
		assertBytes(t, []byte{0b10000000}, bf.ToBytes())

		bf.SetBit(12, false)
		assertBytes(t, []byte{0b10000000, 0b00000000}, bf.ToBytes())

		bf.SetBit(11, true)
		assertBytes(t, []byte{0b10000000, 0b00010000}, bf.ToBytes())

		bf.SetBit(8, true)
		assertBytes(t, []byte{0b10000000, 0b10010000}, bf.ToBytes())
	})

	t.Run("String() returns a human readable representation", func(t *testing.T) {
		input := ds.NewBitfieldFromBits(0, 1, 1, 0, 1, 0, 1, 1, 1)
		got := input.String()
		want := "0110101110000000"
		helpers.AssertEqual(t, want, got)
	})
}

func TestBitfield_GetBit(t *testing.T) {
	t.Run("returns correct boolean value for bits", func(t *testing.T) {
		bf := *ds.NewBitfieldFromBits(1, 0, 1, 1, 0, 0, 1, 0, 1)

		expected := []bool{true, false, true, true, false, false, true, false, true}
		for i, want := range expected {
			helpers.AssertEqual(t, want, bf.GetBit(uint(i)))
		}
	})

	t.Run("returns false when index exceeds allocated bytes", func(t *testing.T) {
		bf := *ds.NewBitfield([]byte{0b10000000})

		helpers.AssertEqual(t, true, bf.GetBit(0))
		helpers.AssertEqual(t, false, bf.GetBit(1))
		helpers.AssertEqual(t, false, bf.GetBit(7))
		helpers.AssertEqual(t, false, bf.GetBit(8))
		helpers.AssertEqual(t, false, bf.GetBit(100))
	})
}

func assertBytes(t testing.TB, want, got []byte) {
	t.Helper()

	if !slices.Equal(got, want) {
		gotDisplay := make([]string, len(got))
		wantDisplay := make([]string, len(want))

		for i := range len(got) {
			gotDisplay[i] = fmt.Sprintf("%08b", got[i])
		}
		for i := range len(want) {
			wantDisplay[i] = fmt.Sprintf("%08b", want[i])
		}

		t.Errorf("expected %v, got %v", wantDisplay, gotDisplay)
	}
}
