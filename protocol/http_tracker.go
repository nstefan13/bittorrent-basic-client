package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-viper/mapstructure/v2"
	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/serde"
)

// HTTPTracker maintains the state required to communicate with an HTTP/HTTPS tracker.
// Each instance is bound to a single torrent's info_hash, so distinct instances
// must be instantiated for different torrents even if they announce to the same tracker URL.
type HTTPTracker struct {
	TrackerURL url.URL
	PeerID     [20]byte
	InfoHash   [20]byte

	// ListeningPort is the TCP port on which our local BitTorrent client listens for incoming
	// peer connections. The tracker advertises this port to other peers in the swarm so they can connect back to us.
	ListeningPort uint16
}

// NewHTTPTracker initializes an [HTTPTracker] configured for a specific torrent info_hash
// and local client listening port. It generates a randomized 20-byte peer ID for announcements.
func NewHTTPTracker(trackerURL url.URL, infoHash [20]byte, listeningPort uint16) *HTTPTracker {
	peerId := helpers.SHA1String(
		trackerURL.String(),
		strconv.Itoa(int(time.Now().Unix())),
		strconv.Itoa(rand.Int()),
	)

	return &HTTPTracker{
		TrackerURL:    trackerURL,
		PeerID:        peerId,
		InfoHash:      infoHash,
		ListeningPort: listeningPort,
	}
}

// RequestPeers sends an announce request to the tracker and returns the list of active peers.
//
// Error conditions:
//   - Returns [ErrTrackerReportedFailure] if the tracker explicitly rejected the request via "failure reason".
//   - Returns [ErrMalformedTrackerResponse] if the tracker returned invalid bencode or a response that violates the BEP3 specification.
//   - Returns network or I/O errors encountered while communicating with the tracker.
func (t *HTTPTracker) RequestPeers() ([]PeerAddress, error) {
	request := t.TrackerURL
	q := request.Query()
	q.Set("info_hash", string(t.InfoHash[:]))
	q.Set("peer_id", string(t.PeerID[:]))
	q.Set("port", strconv.Itoa(int(t.ListeningPort)))
	q.Set("event", "started")
	request.RawQuery = q.Encode()

	resp, err := http.Get(request.String())
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bdict, err := serde.Decode[bcd.BDict](body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedTrackerResponse, err)
	}

	failureReason, err := serde.ExtractValueFromBDict[bcd.BString](bdict, "failure reason")
	if err == nil {
		return nil, fmt.Errorf("%w: %q", ErrTrackerReportedFailure, string(failureReason))
	}
	if errors.Is(err, serde.ErrTypeMismatch) {
		var zero bcd.BString
		return nil, fmt.Errorf("%w: %w: expected 'failure reason' to be a %T, instead it is a %T", ErrMalformedTrackerResponse, serde.ErrTypeMismatch, zero, bdict["failure reason"])
	}

	untypedPeers, err := serde.ExtractValueFromBDict[bcd.BString](bdict, "peers")
	switch {
	case err == nil:
		// if peers is a string, then it must be the compact representation
		return parseCompactPeerList([]byte(untypedPeers))

	case errors.Is(err, serde.ErrTypeMismatch):
		// if peers is not a string (which indicates a compact representation), then
		// it must be a dictionary (the loose representation)
		untypedPeers, err := serde.ExtractValueFromBDict[bcd.BList](bdict, "peers")
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMalformedTrackerResponse, err)
		}
		peers, err := parseLoosePeerList(untypedPeers)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrMalformedTrackerResponse, err)
		}
		return peers, nil

	default:
		return nil, fmt.Errorf("%w: %w", ErrMalformedTrackerResponse, err)
	}
}

func parseLoosePeerList(data bcd.BList) ([]PeerAddress, error) {
	var peers []struct {
		IP   string `mapstructure:"ip"`
		Port uint16 `mapstructure:"port"`
	}

	config := &mapstructure.DecoderConfig{
		Result:     &peers,
		ErrorUnset: true,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		log.Fatalf("parseLoosePeerList: Error occurred while creating the DecoderConfig: %v", err)
	}

	err = decoder.Decode(data)
	if err != nil {
		return nil, err
	}

	var result = make([]PeerAddress, len(peers))
	for index, peer := range peers {
		result[index].Ip = net.ParseIP(peer.IP)
		result[index].Port = peer.Port
	}

	return result, nil
}

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
