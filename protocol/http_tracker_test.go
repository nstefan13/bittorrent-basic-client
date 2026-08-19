package protocol_test

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
	protocol "github.com/nstefan13/bittorrent/protocol"
	"github.com/nstefan13/bittorrent/serde"
)

func TestHTTPTrackerRequestPeers(t *testing.T) {
	t.Run("sends valid announce request parameters", func(t *testing.T) {
		env := setup(bcd.BDict{
			"peers": bcd.BString([]byte{}),
		})
		defer env.TrackerServer.Close()

		env.TrackerClient.RequestPeers()

		helpers.AssertEqual(t, http.MethodGet, env.TrackerHandler.Method)
		helpers.AssertEqual(t, string(env.TrackerClient.InfoHash[:]), env.TrackerHandler.InfoHash)
		helpers.AssertEqual(t, string(env.TrackerClient.PeerID[:]), env.TrackerHandler.PeerId)
		helpers.AssertEqual(t, env.TrackerClient.ListeningPort, env.TrackerHandler.Port)
		helpers.AssertEqual(t, "started", env.TrackerHandler.Event)
	})

	t.Run("parses the compact peer list response", func(t *testing.T) {
		env := setup(bcd.BDict{
			"peers": bcd.BString([]byte{
				100, 54, 76, 99, 0x13, 0x88,
				192, 168, 1, 99, 0x0B, 0xB8,
				121, 192, 239, 89, 0x00, 0x16,
			}),
		})
		defer env.TrackerServer.Close()

		got, err := env.TrackerClient.RequestPeers()
		want := []protocol.PeerAddress{
			protocol.NewPeerAddress("100.54.76.99", 5000),
			protocol.NewPeerAddress("192.168.1.99", 3000),
			protocol.NewPeerAddress("121.192.239.89", 22),
		}

		helpers.AssertNoError(t, err)
		helpers.AssertEqual(t, want, got)
	})

	t.Run("parses the loose peer list response", func(t *testing.T) {
		env := setup(bcd.BDict{
			"peers": bcd.BList{
				bcd.BDict{
					"peer id": bcd.BString(randomHashAsString()),
					"ip":      bcd.BString("44.191.67.194"),
					"port":    bcd.BInt(3456),
				},
				bcd.BDict{
					"peer id": bcd.BString(randomHashAsString()),
					"ip":      bcd.BString("32.77.239.68"),
					"port":    bcd.BInt(7389),
				},
			},
		})
		defer env.TrackerServer.Close()

		got, err := env.TrackerClient.RequestPeers()
		want := []protocol.PeerAddress{
			protocol.NewPeerAddress("44.191.67.194", 3456),
			protocol.NewPeerAddress("32.77.239.68", 7389),
		}

		helpers.AssertNoError(t, err)
		helpers.AssertEqual(t, want, got)
	})
}

func TestHTTPTrackerRequestPeersErrors(t *testing.T) {
	testCases := map[string]struct {
		Response      []byte
		ExpectedError error
	}{
		"invalid data returned from tracker": {
			Response:      []byte("akjdsfl;asdhf;lh"),
			ExpectedError: protocol.ErrMalformedTrackerResponse,
		},
		"invalid type for 'failure reason'": {
			Response:      mustEncode(bcd.BDict{"failure reason": bcd.BInt(232)}),
			ExpectedError: protocol.ErrMalformedTrackerResponse,
		},
		"tracker reported failure": {
			Response:      mustEncode(bcd.BDict{"failure reason": bcd.BString("requested download is not authorized")}),
			ExpectedError: protocol.ErrTrackerReportedFailure,
		},
		"missing 'peers' key": {
			Response:      mustEncode(bcd.BDict{"interval": bcd.BInt(1800)}),
			ExpectedError: protocol.ErrMalformedTrackerResponse,
		},
		"invalid entry in loose peer list": {
			Response: mustEncode(bcd.BDict{
				"peers": bcd.BList{
					bcd.BDict{
						"peer id": bcd.BString(randomHashAsString()),
						// missing "ip" and "port"
					},
				},
			}),
			ExpectedError: protocol.ErrMalformedTrackerResponse,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			env := setupRaw(tc.Response)
			defer env.TrackerServer.Close()

			_, err := env.TrackerClient.RequestPeers()
			helpers.AssertError(t, err, tc.ExpectedError)
		})
	}
}

func mustEncode(data any) []byte {
	buffer, err := serde.Encode(data)
	if err != nil {
		panic(err)
	}
	return buffer
}

type stubTrackerHandler struct {
	// The unique reponse that will be served to the client
	Response []byte

	// Metadata regarding the client's request
	Method   string // HTTP request method used by the client (e.g. "GET")
	InfoHash string // Raw info_hash query parameter from announce request
	PeerId   string // Client's advertised peer_id query parameter
	Port     int    // Client's listening port (-1 if missing or invalid integer)
	Event    string // Announce event query parameter (e.g. "started", "completed", "stopped", empty string)
}

func (s *stubTrackerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func newStubTrackerHandlerWithRawResponse(response []byte) *stubTrackerHandler {
	handler := &stubTrackerHandler{
		Response: response,
	}
	return handler
}

func newStubTrackerHandler(response bcd.BDict) *stubTrackerHandler {
	buffer, _ := serde.Encode(response)
	return newStubTrackerHandlerWithRawResponse(buffer)
}

type setupResult struct {
	TrackerHandler *stubTrackerHandler
	TrackerServer  *httptest.Server
	TrackerURL     *url.URL
	TrackerClient  *protocol.HTTPTracker
}

func setupRaw(data []byte) setupResult {
	trackerHandler := newStubTrackerHandlerWithRawResponse(data)
	trackerServer := httptest.NewServer(trackerHandler)
	trackerURL, _ := url.Parse(trackerServer.URL)

	infoHash := randomHash()
	listeningPort := uint16(rand.IntN(65536))
	trackerClient := protocol.NewHTTPTracker(*trackerURL, infoHash, listeningPort)

	return setupResult{
		TrackerHandler: trackerHandler,
		TrackerServer:  trackerServer,
		TrackerURL:     trackerURL,
		TrackerClient:  trackerClient,
	}
}

func setup(requestData bcd.BDict) setupResult {
	trackerHandler := newStubTrackerHandler(requestData)
	trackerServer := httptest.NewServer(trackerHandler)
	trackerURL, _ := url.Parse(trackerServer.URL)

	infoHash := randomHash()
	listeningPort := uint16(rand.IntN(65536))
	trackerClient := protocol.NewHTTPTracker(*trackerURL, infoHash, listeningPort)

	return setupResult{
		TrackerHandler: trackerHandler,
		TrackerServer:  trackerServer,
		TrackerURL:     trackerURL,
		TrackerClient:  trackerClient,
	}
}

func randomHash() [20]byte {
	return helpers.SHA1String(strconv.Itoa(rand.Int()))
}

func randomHashAsString() string {
	slice := randomHash()
	return string(slice[:])
}
