package serde_test

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/serde"
)

func TestDeserializePacket(t *testing.T) {
	testCases := []struct {
		name           string
		input          []byte
		want           any
		expectedErrors []error
	}{
		// GENERIC ERRORS
		{
			name:           "packet too big",
			input:          []byte{0xff, 0xff, 0xff, 0xff},
			want:           nil,
			expectedErrors: []error{serde.ErrPacketTooBig},
		},
		{
			name:           "unknown packet-id",
			input:          []byte{0x00, 0x00, 0x00, 0x01, 0xff},
			want:           nil,
			expectedErrors: []error{serde.ErrUnsupportedPacketType},
		},
		{
			name:           "stream cut off while reading the length",
			input:          []byte{0xff, 0xff, 0xff},
			want:           nil,
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},
		{
			name:           "stream cut off while reading type-id",
			input:          []byte{0x00, 0x00, 0x00, 0x01},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},

		// KEEP ALIVE PACKET
		{
			name:           "keep-alive packet",
			input:          []byte{0, 0, 0, 0},
			want:           serde.KeepAlivePacket{},
			expectedErrors: nil,
		},

		// CHOKE PACKET
		{
			name:           "choke packet",
			input:          []byte{0, 0, 0, 1, 0},
			want:           serde.ChokePacket{},
			expectedErrors: nil,
		},

		// UNCHOKE PACKET
		{
			name:           "unchoke packet",
			input:          []byte{0, 0, 0, 1, 1},
			want:           serde.UnchokePacket{},
			expectedErrors: nil,
		},

		// INTERESTED PACKET
		{
			name:           "interested packet",
			input:          []byte{0, 0, 0, 1, 2},
			want:           serde.InterestedPacket{},
			expectedErrors: nil,
		},

		// NOT INTERESTED PACKET
		{
			name:           "not interested packet",
			input:          []byte{0, 0, 0, 1, 3},
			want:           serde.NotInterestedPacket{},
			expectedErrors: nil,
		},

		// HAVE PACKET
		{
			name:           "have packet",
			input:          []byte{0, 0, 0, 5, 4, 0, 0, 0, 9},
			want:           serde.HavePacket{9},
			expectedErrors: nil,
		},
		{
			name:           "have packet: length must be 5",
			input:          []byte{0, 0, 0, 3, 4, 0, 0, 0, 1},
			want:           nil,
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name:           "have packet: stream cut off while reading the piece index",
			input:          []byte{0, 0, 0, 5, 4, 0, 0},
			want:           nil,
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},

		// BITFIELD PACKET
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
			name:           "bitfield packet: invalid length of 1 byte",
			input:          []byte{0x00, 0x00, 0x00, 0x01, 0x05},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name:           "bitfield packet: stream cut off while reading payload",
			input:          []byte{0x00, 0x00, 0x00, 0x02, 0x05},
			want:           newBitfieldPacket(""),
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},

		// REQUEST PACKET
		{
			name: "request packet",
			input: []byte{0x00, 0x00, 0x00, 0x0d, 6,
				0, 0, 0, 1,
				0, 0, 0, 0,
				0, 0, 0x40, 0x00},
			want:           serde.RequestPacket{1, 0, 0x4000},
			expectedErrors: nil,
		},
		{
			name: "request packet: length must be 13",
			input: []byte{0x00, 0x00, 0x00, 0x05, 6,
				0, 0, 0, 1},
			want:           nil,
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name: "request packet: stream cut off while reading payload",
			input: []byte{0x00, 0x00, 0x00, 0x0d, 6,
				0, 0, 0, 1,
				0, 0},
			want:           nil,
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},

		// PIECE PACKET
		{
			name: "piece packet",
			input: []byte{0x00, 0x00, 0x00, 0x0d, 7,
				0, 0, 0, 1,
				0, 0, 0, 0,
				0xde, 0xad, 0xbe, 0xef},
			want: serde.PiecePacket{
				PieceIndex:  1,
				BeginOffset: 0,
				Block:       []byte{0xde, 0xad, 0xbe, 0xef},
			},
			expectedErrors: nil,
		},
		{
			name:           "piece packet: invalid length of 1 byte",
			input:          []byte{0x00, 0x00, 0x00, 0x01, 7},
			want:           nil,
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name:           "piece packet: invalid length of 9 bytes",
			input:          []byte{0x00, 0x00, 0x00, 0x09, 7},
			want:           nil,
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name: "piece packet: stream cut off while reading payload",
			input: []byte{0x00, 0x00, 0x00, 0x0d, 7,
				0, 0, 0, 1,
				0, 0},
			want:           nil,
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},

		// CANCEL PACKET
		{
			name: "cancel packet",
			input: []byte{0x00, 0x00, 0x00, 0x0d, 8,
				0, 0, 0, 1,
				0, 0, 0, 0,
				0, 0, 0x40, 0x00},
			want:           serde.CancelPacket{1, 0, 0x4000},
			expectedErrors: nil,
		},
		{
			name: "cancel packet: length must be 13",
			input: []byte{0x00, 0x00, 0x00, 0x05, 8,
				0, 0, 0, 1},
			want:           nil,
			expectedErrors: []error{serde.ErrMalformedPacket},
		},
		{
			name: "cancel packet: stream cut off while reading payload",
			input: []byte{0x00, 0x00, 0x00, 0x0d, 8,
				0, 0, 0, 1,
				0, 0},
			want:           nil,
			expectedErrors: []error{io.ErrUnexpectedEOF},
		},
	}

	for index, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("test #%d", index)
			got, err := serde.DeserializePacket(bytes.NewReader(tc.input))

			if tc.expectedErrors == nil {
				helpers.AssertNoError(t, err)
				assertPackets(t, tc.want, got)
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

func assertPackets(t testing.TB, want, got any) {
	if !reflect.DeepEqual(got, want) {
		message := fmt.Sprintf("expected %#v, got %#v", want, got)
		message = strings.ReplaceAll(message, "serde.", "")
		t.Error(message)
	}
}
