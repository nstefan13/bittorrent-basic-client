package protocol_test

import (
	"testing"

	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/parsers"
	"github.com/nstefan13/bittorrent/protocol"
)

type StubTracker struct{}

func (t *StubTracker) RequestPeers(infoDict parsers.InfoDict) {}

func TestRequestPeers(t *testing.T) {
	infoDict := parsers.InfoDict{}
	tracker := &StubTracker{}

	tracker.RequestPeers(infoDict)
}

func TestNewTrackerFromMetaInfo(t *testing.T) {
	t.Run("choose the HTTPTracker for an http URL", func(t *testing.T) {
		metainfo := metainfoWithUrl("http://tracker.opentrackr.org:1337/announce")
		tracker, err := protocol.NewTrackerFromMetaInfo(metainfo)
		helpers.AssertNoError(t, err)
		assertTrackerType[*protocol.HTTPTracker](t, tracker)
	})

	t.Run("choose the HTTPTracker for an https URL", func(t *testing.T) {
		metainfo := metainfoWithUrl("https://tracker.tamersveil.org:443/announce")
		tracker, err := protocol.NewTrackerFromMetaInfo(metainfo)
		helpers.AssertNoError(t, err)
		assertTrackerType[*protocol.HTTPTracker](t, tracker)
	})

	t.Run("return ErrUnsupportedTracker for a UDP URL", func(t *testing.T) {
		metainfo := metainfoWithUrl("udp://open.demonii.com:1337/announce")
		_, err := protocol.NewTrackerFromMetaInfo(metainfo)
		helpers.AssertError(t, err, protocol.ErrUnsupportedTracker)
	})
}

func metainfoWithUrl(url string) *parsers.MetaInfo {
	return &parsers.MetaInfo{
		TrackerURL: helpers.UrlFromString(url),
	}
}

func assertTrackerType[E any](t testing.TB, actualTracker any) {
	t.Helper()
	if _, ok := actualTracker.(*protocol.HTTPTracker); !ok {
		t.Errorf("expected the tracker to be of type %T, instead it is of type %T", (*protocol.HTTPTracker)(nil), actualTracker)
	}
}
