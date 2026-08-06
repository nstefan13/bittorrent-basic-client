package protocol

import (
	"errors"

	"github.com/nstefan13/bittorrent/parsers"
)

var ErrUnsupportedTracker = errors.New("unsupported tracker")

type Tracker interface {
	RequestPeers(parsers.InfoDict)
}

// NewTrackerFromMetaInfo constructs a Tracker suitable for communicating with
// the tracker specified in the metainfo's announce URL.
//
// It returns an [HTTPTracker] for "http" or "https" schemes. If the scheme is unsupported,
// it returns nil and [ErrUnsupportedTracker].
func NewTrackerFromMetaInfo(metaInfo *parsers.MetaInfo) (Tracker, error) {
	scheme := metaInfo.TrackerURL.Scheme

	if scheme == "http" || scheme == "https" {
		return &HTTPTracker{}, nil
	}

	return nil, ErrUnsupportedTracker
}
