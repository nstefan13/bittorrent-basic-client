package protocol_test

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/serde"
	protocol "github.com/nstefan13/bittorrent/protocol"
)

type StubTrackerHandler struct {
	// The unique reponse that will be served to the client
	Response []byte

	// Metadata regarding the client's request
	Method   string // HTTP request method used by the client (e.g. "GET")
	InfoHash string // Raw info_hash query parameter from announce request
	PeerId   string // Client's advertised peer_id query parameter
	Port     int    // Client's listening port (-1 if missing or invalid integer)
	Event    string // Announce event query parameter (e.g. "started", "completed", "stopped", empty string)
}

func (s *StubTrackerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	// Store the metadata
	s.Method = r.Method

	aux1 := r
	aux2 := aux1.URL
	aux3 := aux2.Query()
	aux4 := aux3.Get("info_hash")
	s.InfoHash = aux4

	s.PeerId = r.URL.Query().Get("peer_id")
	s.Event = r.URL.Query().Get("event")
	s.Port, err = strconv.Atoi(r.URL.Query().Get("port"))
	if err != nil {
		s.Port = -1
	}

	// Serve the actual response
	w.Write(s.Response)
}

func NewStubTrackerHandler(response bencoding.BDict) *StubTrackerHandler {
	buffer, _ := serde.Encode(response)
	handler := &StubTrackerHandler{
		Response: buffer,
	}
	return handler
}

func TestHTTPTrackerRequestPeers(t *testing.T) {
	t.Run("sends valid announce request parameters", func(t *testing.T) {
		trackerHandler := NewStubTrackerHandler(bencoding.BDict{
			"peers": bencoding.BString([]byte{}),
		})
		trackerServer := httptest.NewServer(trackerHandler)
		trackerURL, _ := url.Parse(trackerServer.URL)

		infoHash := randomInfoHash()
		listeningPort := uint16(rand.IntN(65536))
		trackerClient := protocol.NewHTTPTracker(*trackerURL, infoHash, listeningPort)

		trackerClient.RequestPeers()

		helpers.AssertEqual(t, http.MethodGet, trackerHandler.Method)
		helpers.AssertEqual(t, string(infoHash[:]), trackerHandler.InfoHash)
		helpers.AssertEqual(t, string(trackerClient.PeerID[:]), trackerHandler.PeerId)
		helpers.AssertEqual(t, listeningPort, trackerHandler.Port)
		helpers.AssertEqual(t, "started", trackerHandler.Event)
	})

	t.Run("parses the compact peer list response", func(t *testing.T) {
		trackerHandler := NewStubTrackerHandler(bencoding.BDict{
			"peers": bencoding.BString([]byte{
				100, 54, 76, 99, 0x13, 0x88,
				192, 168, 1, 99, 0x0B, 0xB8,
				121, 192, 239, 89, 0x00, 0x16,
			}),
		})
		trackerServer := httptest.NewServer(trackerHandler)
		trackerURL, _ := url.Parse(trackerServer.URL)

		infoHash := randomInfoHash()
		listeningPort := uint16(rand.IntN(65536))
		trackerClient := protocol.NewHTTPTracker(*trackerURL, infoHash, listeningPort)

		got := trackerClient.RequestPeers()
		want := []protocol.PeerAddress{
			protocol.NewPeerAddress("100.54.76.99", 5000),
			protocol.NewPeerAddress("192.168.1.99", 3000),
			protocol.NewPeerAddress("121.192.239.89", 22),
		}

		helpers.AssertEqual(t, want, got)
	})
}

func randomInfoHash() [20]byte {
	return helpers.SHA1String(strconv.Itoa(rand.Int()))
}
