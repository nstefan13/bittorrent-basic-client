package serde_test

import (
	"testing"

	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/serde"
)

func TestSerializeObjects(t *testing.T) {
	got := serde.SerializeObjects(
		serde.KeepAlivePacket{},
		BitfieldPacketFromString("100101101001101"),
		serde.RequestPacket{
			PieceIndex:  1,
			BeginOffset: 0,
			Length:      0x4000,
		},
	)

	want := []byte{
		0, 0, 0, 0,
		0x00, 0x00, 0x00, 0x03, 5, 0b10010110, 0b10011010,
		0x00, 0x00, 0x00, 0x0d, 6,
		0, 0, 0, 1,
		0, 0, 0, 0,
		0, 0, 0x40, 0x00,
	}

	helpers.AssertEqual(t, want, got)
}
