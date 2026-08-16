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

		return BitfieldPacket{*ds.NewBitfield(payload)}, nil

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

func (p KeepAlivePacket) Serialize() []byte {
	return []byte{0, 0, 0, 0}
}

func (p ChokePacket) Serialize() []byte {
	return []byte{0, 0, 0, 1, 0}
}

func (p UnchokePacket) Serialize() []byte {
	return []byte{0, 0, 0, 1, 1}
}

func (p InterestedPacket) Serialize() []byte {
	return []byte{0, 0, 0, 1, 2}
}

func (p NotInterestedPacket) Serialize() []byte {
	return []byte{0, 0, 0, 1, 3}
}

func (p HavePacket) Serialize() []byte {
	buf := make([]byte, 9)
	binary.BigEndian.PutUint32(buf[0:4], 5)
	buf[4] = 4
	binary.BigEndian.PutUint32(buf[5:9], uint32(p.PieceIndex))
	return buf
}

func (p BitfieldPacket) Serialize() []byte {
	raw := p.Bitfield.ToBytes()
	buf := make([]byte, 5+len(raw))
	binary.BigEndian.PutUint32(buf[0:4], uint32(1+len(raw)))
	buf[4] = 5
	copy(buf[5:], raw)
	return buf
}

func (p RequestPacket) Serialize() []byte {
	buf := make([]byte, 17)
	binary.BigEndian.PutUint32(buf[0:4], 13)
	buf[4] = 6
	binary.BigEndian.PutUint32(buf[5:9], uint32(p.PieceIndex))
	binary.BigEndian.PutUint32(buf[9:13], uint32(p.BeginOffset))
	binary.BigEndian.PutUint32(buf[13:17], uint32(p.Length))
	return buf
}

func (p PiecePacket) Serialize() []byte {
	buf := make([]byte, 13+len(p.Block))
	binary.BigEndian.PutUint32(buf[0:4], uint32(9+len(p.Block)))
	buf[4] = 7
	binary.BigEndian.PutUint32(buf[5:9], uint32(p.PieceIndex))
	binary.BigEndian.PutUint32(buf[9:13], uint32(p.BeginOffset))
	copy(buf[13:], p.Block)
	return buf
}

func (p CancelPacket) Serialize() []byte {
	buf := make([]byte, 17)
	binary.BigEndian.PutUint32(buf[0:4], 13)
	buf[4] = 8
	binary.BigEndian.PutUint32(buf[5:9], uint32(p.PieceIndex))
	binary.BigEndian.PutUint32(buf[9:13], uint32(p.BeginOffset))
	binary.BigEndian.PutUint32(buf[13:17], uint32(p.Length))
	return buf
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
