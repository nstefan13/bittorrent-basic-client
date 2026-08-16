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
var ErrPacketTooBig = errors.New("packet is too large")
var ErrUnsupportedPacketType = errors.New("packet type is not supported")

// Rule 1: Packets are Read-Only. They store the received data from the network
// Rule 2: It should be possible for packets to be created only via their constructor (without a New* function)
// Rule 3: Packets should contain only the necessary memebers to store data from the network (so no length or type)

type KeepAlivePacket struct{}

type ChokePacket struct{}

type UnchokePacket struct{}

type InterestedPacket struct{}

type NotInterestedPacket struct{}

type BitfieldPacket struct {
	Bitfield ds.Bitfield
}

type HavePacket struct {
	PieceIndex int
}

type RequestPacket struct {
	PieceIndex  int
	BeginOffset int
	Length      int
}

type PiecePacket struct {
	PieceIndex  int
	BeginOffset int
	Block       []byte
}

type CancelPacket RequestPacket // these two are the same

func DeserializePacket(reader io.Reader) (packet any, err error) {
	serializedLength, err := helpers.ReadExactly(4, reader)
	if err != nil {
		return nil, err
	}

	length, err := validateLength(binary.BigEndian.Uint32(serializedLength))
	if err != nil {
		return nil, err
	}

	// the only packet of length 0 is KeepAlive
	if length == 0 {
		return KeepAlivePacket{}, nil
	}

	buffer, err := helpers.ReadExactly(1, reader)
	if err != nil {
		return nil, err
	}
	packetType := buffer[0]

	switch packetType {
	// CHOKE PACKET
	case 0:
		return ChokePacket{}, nil

	// CHOKE PACKET
	case 1:
		return UnchokePacket{}, nil

	// INTERESTED PACKET
	case 2:
		return InterestedPacket{}, nil

	// NOT INTERESTED PACKET
	case 3:
		return NotInterestedPacket{}, nil

	// HAVE PACKET
	case 4:
		if length != 5 {
			return nil, detailedMalformedPacketError("HAVE packets must have a <length> of 5")
		}

		payload, err := helpers.ReadExactly(4, reader)
		if err != nil {
			return nil, err
		}

		pieceIndex := int(binary.BigEndian.Uint32(payload))

		return HavePacket{pieceIndex}, nil

	// BITFIELD PACKET
	case 5:
		if length < 2 {
			return nil, detailedMalformedPacketError("<length> must be >= 2, got %d instead", length)
		}

		payload, err := helpers.ReadExactly(length-1, reader)
		if err != nil {
			return nil, err
		}

		return BitfieldPacket{*ds.NewBitfieldFromBytes(payload)}, nil

	// REQUEST PACKET
	case 6:
		if length != 13 {
			return nil, detailedMalformedPacketError("REQUEST packets must have a <length> of 13")
		}

		payload, err := helpers.ReadExactly(12, reader)
		if err != nil {
			return nil, err
		}

		return RequestPacket{
			PieceIndex:  int(binary.BigEndian.Uint32(payload[0:])),
			BeginOffset: int(binary.BigEndian.Uint32(payload[4:])),
			Length:      int(binary.BigEndian.Uint32(payload[8:])),
		}, nil

	// PIECE PACKET
	case 7:
		if length <= 9 {
			return nil, detailedMalformedPacketError("PIECE packets must have a <length> strictly greater than 9")
		}

		payload, err := helpers.ReadExactly(length-1, reader)
		if err != nil {
			return nil, err
		}

		return PiecePacket{
			PieceIndex:  int(binary.BigEndian.Uint32(payload[0:])),
			BeginOffset: int(binary.BigEndian.Uint32(payload[4:])),
			Block:       payload[8:],
		}, nil

	// CANCEL PACKET
	case 8:
		if length != 13 {
			return nil, detailedMalformedPacketError("CANCEL packets must have a <length> of 13")
		}

		payload, err := helpers.ReadExactly(12, reader)
		if err != nil {
			return nil, err
		}

		return CancelPacket{
			PieceIndex:  int(binary.BigEndian.Uint32(payload[0:])),
			BeginOffset: int(binary.BigEndian.Uint32(payload[4:])),
			Length:      int(binary.BigEndian.Uint32(payload[8:])),
		}, nil

	default:
		return nil, fmt.Errorf("%w: received <type> %d", ErrUnsupportedPacketType, packetType)
	}

}

func validateLength(length uint32) (int, error) {
	// realistically, the largest packet will have
	// a payload of 16MB, plus some protocol overhead
	// (that +100 is some little wiggle room)
	if length > 16*1024*1024+9+100 {
		return 0, fmt.Errorf("%w: <length> must not exceed ~16MB, got a length of %d bytes", ErrPacketTooBig, length)
	}

	return int(length), nil
}

func detailedMalformedPacketError(format string, a ...any) error {
	result := fmt.Sprintf(format, a...)
	return fmt.Errorf("%w: %s", ErrMalformedPacket, result)
}
