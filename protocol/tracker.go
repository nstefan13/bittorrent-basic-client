package protocol

import (
	"errors"
	"net"

	"github.com/nstefan13/bittorrent/serde"
)

var ErrUnsupportedTracker = errors.New("unsupported tracker")

type PeerAddress struct {
	Ip   net.IP
	Port uint16
}

func NewPeerAddress(ip string, port uint16) PeerAddress {
	return PeerAddress{
		Ip:   net.ParseIP(ip),
		Port: port,
	}
}

type Tracker interface {
	RequestPeers() []PeerAddress
}

// NewTrackerFromMetaInfo constructs a Tracker suitable for communicating with
// the tracker specified in the metainfo's announce URL.
//
// It returns an [HTTPTracker] for "http" or "https" schemes. If the scheme is unsupported,
// it returns nil and [ErrUnsupportedTracker].
//
// Note:
// A Tracker object stores all the required information to do VALID requests to a tracker.
// Since the BEP3 specification mandates that a listening port (`port`) must be provided in any valid
// tracker request, this function also requires a port, even though one might just use the return value
// to only get a list of peers in a download-only flow. If that's the case, one could simply supply a dummy
// port like 6881 or some random value and move on.
func NewTrackerFromMetaInfo(metaInfo *serde.MetaInfo, listeningPort uint16) (Tracker, error) {
	scheme := metaInfo.TrackerURL.Scheme

	if scheme == "http" || scheme == "https" {
		return NewHTTPTracker(metaInfo.TrackerURL, metaInfo.Info.InfoHash(), uint16(listeningPort)), nil
	}

	return nil, ErrUnsupportedTracker
}
