package protocol

import (
	"encoding/binary"
	"errors"
	"net"
)

var ErrMalformedTrackerResponse = errors.New("tracker returned invalid response")

func parseCompactPeerList(compactList []byte) ([]PeerAddress, error) {
	if len(compactList)%6 != 0 || len(compactList) == 0 {
		// TODO: return custom error message
		// the caller should wrap it in a ErrMalformedTrackerResponse, not us
		return nil, ErrMalformedTrackerResponse
	}

	peers := make([]PeerAddress, 0, len(compactList)/6)
	for i := 0; i < len(compactList); i += 6 {
		rawIP := compactList[i : i+4]
		rawPort := compactList[i+4 : i+6]

		ip := net.IPv4(rawIP[0], rawIP[1], rawIP[2], rawIP[3])
		port := binary.BigEndian.Uint16(rawPort)

		peers = append(peers, PeerAddress{
			Ip:   ip,
			Port: port,
		})
	}

	return peers, nil
}
