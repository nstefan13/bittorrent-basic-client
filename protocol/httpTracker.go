package protocol

import "github.com/nstefan13/bittorrent/parsers"

type HTTPTracker struct{}

func (t *HTTPTracker) RequestPeers(parsers.InfoDict) {}
