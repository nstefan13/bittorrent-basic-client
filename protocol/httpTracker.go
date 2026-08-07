package protocol

import (
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"time"

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

func (t *HTTPTracker) RequestPeers() []PeerAddress {
	request := t.TrackerURL
	q := request.Query()
	q.Set("info_hash", string(t.InfoHash[:]))
	q.Set("peer_id", string(t.PeerID[:]))
	q.Set("port", strconv.Itoa(int(t.ListeningPort)))
	q.Set("event", "started")
	request.RawQuery = q.Encode()

	resp, _ := http.Get(request.String())

	body, _ := io.ReadAll(resp.Body)

	bdict, _ := serde.Decode[bcd.BDict](body)

	rawPeers, _ := serde.ExtractValueFromBDict[bcd.BString](bdict, "peers")

	peers, _ := parseCompactPeerList([]byte(rawPeers))

	return peers
}

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
