package serde

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/helpers"
)

var ErrMalformedPacket = errors.New("malformed peer-wire-protocol packet")

// Rule 1: Packets are Read-Only. They store the received data from the network
// Rule 2: It should be possible for packets to be created only via their constructor (without a New* function)
// Rule 3: Packets should contain only the necessary memebers to store data from the network (so no length or type)
type BitfieldPacket struct {
	Bitfield ds.Bitfield
}

func DeserializeBitfield(reader io.Reader) (BitfieldPacket, error) {
	var zero BitfieldPacket

	buffer, err := helpers.ReadExactly(4, reader)
	if err != nil {
		return zero, err
	}

	payloadLength := int(binary.BigEndian.Uint32(buffer)) - 1
	if payloadLength <= 0 {
		return zero, detailedMalformedPacketError("<length> must be greater than 1, got %d instead", payloadLength+1)
	}

	buffer, err = helpers.ReadExactly(1, reader)
	if err != nil {
		return zero, err
	}

	packetType := buffer[0]
	if packetType != 5 {
		return zero, detailedMalformedPacketError("<type> must be 5 for a Bitfield Packet, not %d", packetType)
	}

	payload, err := helpers.ReadExactly(int(payloadLength), reader)
	if err != nil {
		return zero, err
	}

	return BitfieldPacket{*ds.NewBitfieldFromBytes(payload)}, nil
}

func detailedMalformedPacketError(format string, a ...any) error {
	result := fmt.Sprintf(format, a...)
	return fmt.Errorf("%w: %s", ErrMalformedPacket, result)
}
