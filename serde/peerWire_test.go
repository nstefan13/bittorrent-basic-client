package serde_test

import (
	"bytes"
	"io"
	"testing"

	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/serde"
)

func TestSerializeBitfield(t *testing.T) {
	testCases := []struct {
		name           string
		input          []byte
		want           serde.BitfieldPacket
		expectedErrors []error
	}{
		{
			name:           "7 bits bitfield",
			input:          []byte{0x00, 0x00, 0x00, 0x02, 5, 0b00100110},
			want:           newBitfieldPacket("0010011"),
			expectedErrors: nil,
		},
		{
			name:           "9 bits bitfield",
			input:          []byte{0x00, 0x00, 0x00, 0x03, 5, 0b10010110, 0b10000000},
			want:           newBitfieldPacket("100101101"),
			expectedErrors: nil,
		},
		{
			name:           "15 bits bitfield",
			input:          []byte{0x00, 0x00, 0x00, 0x03, 5, 0b10010110, 0b10011010},
			want:           newBitfieldPacket("100101101001101"),
			expectedErrors: nil,
		},
		{
			name:           "payload cut at length",
			input:          []byte{0x00, 0x00},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},
		{
			name:           "invalid length of 0 bytes",
			input:          []byte{0x00, 0x00, 0x00, 0x00},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name:           "invalid length of 1 byte",
			input:          []byte{0x00, 0x00, 0x00, 0x01},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name:           "invalid type for Bitfield Packet",
			input:          []byte{0x00, 0x00, 0x00, 0x02, 4},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name:           "type id cut in the middle of transmission",
			input:          []byte{0x00, 0x00, 0x00, 0x02},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},
		{
			name:           "payload cut in the middle of transmission",
			input:          []byte{0x00, 0x00, 0x00, 0x02, 5},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := serde.DeserializeBitfield(bytes.NewReader(tc.input))

			if tc.expectedErrors == nil {
				helpers.AssertNoError(t, err)
				assertBitfieldPackets(t, tc.want, got)
			} else {
				for _, expectedErr := range tc.expectedErrors {
					helpers.AssertError(t, err, expectedErr)
				}
			}
		})
	}
}

func newBitfieldPacket(description string) serde.BitfieldPacket {
	bf := ds.NewBitfield()

	for index, r := range description {
		if r == '1' {
			bf.SetBit(uint(index), true)
		}
	}

	return serde.BitfieldPacket{*bf}
}

func assertBitfieldPackets(t testing.TB, want, got serde.BitfieldPacket) {
	t.Helper()
	helpers.AssertEqual(t, want.Bitfield.String(), got.Bitfield.String())
}
