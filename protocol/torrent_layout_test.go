package protocol_test

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"

	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/protocol"
	p "github.com/nstefan13/bittorrent/protocol"
	"github.com/nstefan13/bittorrent/serde"
)

func TestInfodictTorrentLayout(t *testing.T) {

	t.Run("single file mode", func(t *testing.T) {
		piecesWant, hashes := piecesSample()

		metainfo := serde.MetaInfo{
			Info: serde.InfoDict{
				Length:      5 * 1024,
				PieceLength: 3 * 1024,
				Pieces:      string(hashes[0][:]) + string(hashes[1][:]),
			},
		}
		blockSize := 2 * 1024
		piecesGot := p.NewTorrentLayout(metainfo, blockSize).Pieces()

		AssertPieces(t, piecesGot, piecesWant)
	})

	t.Run("multiple file mode", func(t *testing.T) {
		piecesWant, hashes := piecesSample()

		metainfo := serde.MetaInfo{
			Info: serde.InfoDict{
				Files: []serde.File{
					{
						Length: 4 * 1024,
					},
					{
						Length: 1024,
					},
				},
				PieceLength: 3 * 1024,
				Pieces:      string(hashes[0][:]) + string(hashes[1][:]),
			},
		}
		blockSize := 2 * 1024
		piecesGot := p.NewTorrentLayout(metainfo, blockSize).Pieces()

		AssertPieces(t, piecesGot, piecesWant)
	})

	t.Run("panics if PieceLength is <= 0", func(t *testing.T) {
		assert.Panics(t, func() {
			metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 100, PieceLength: 0, Pieces: strings.Repeat("0", 20)}}
			p.NewTorrentLayout(metainfo, 16)
		})
	})

	t.Run("panics if Length is <= 0 in single file mode", func(t *testing.T) {
		assert.Panics(t, func() {
			metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 0, PieceLength: 10, Pieces: strings.Repeat("0", 20)}}
			p.NewTorrentLayout(metainfo, 16)
		})
	})

	t.Run("panics if one of the files has <= 0 length in multi file mode", func(t *testing.T) {
		assert.Panics(t, func() {
			metainfo := serde.MetaInfo{Info: serde.InfoDict{
				Files:       []serde.File{{Length: 100}, {Length: -5}},
				PieceLength: 10,
				Pieces:      strings.Repeat("0", 200),
			}}
			p.NewTorrentLayout(metainfo, 16)
		})
	})

	t.Run("panics if block length is <= 0", func(t *testing.T) {
		assert.Panics(t, func() {
			metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 100, PieceLength: 10, Pieces: strings.Repeat("0", 200)}}
			p.NewTorrentLayout(metainfo, 0)
		})
	})

	t.Run("caps block length to piece length if block length > piece length", func(t *testing.T) {
		metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 10, PieceLength: 10, Pieces: strings.Repeat("0", 20)}}
		layout := p.NewTorrentLayout(metainfo, 20).Pieces()
		helpers.AssertEqual(t, 1, layout.Len())
		piece := layout.Get(0)
		helpers.AssertEqual(t, 10, piece.Size)
		helpers.AssertEqual(t, 1, piece.Blocks.Len())
		helpers.AssertEqual(t, 10, piece.Blocks.Get(0).Size)
	})

	t.Run("every piece has at least one block", func(t *testing.T) {
		metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 25, PieceLength: 10, Pieces: strings.Repeat("0", 60)}}
		pieces := p.NewTorrentLayout(metainfo, 4).Pieces()

		for pieceIndex := 0; pieceIndex < pieces.Len(); pieceIndex++ {
			if pieces.Get(pieceIndex).Blocks.Len() < 1 {
				t.Fatalf("piece %d has no blocks", pieceIndex)
			}
		}
	})

	t.Run("panics if Pieces string length is not a multiple of 20", func(t *testing.T) {
		assert.Panics(t, func() {
			metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 100, PieceLength: 10, Pieces: "short"}}
			p.NewTorrentLayout(metainfo, 5)
		})
	})

	t.Run("panics if Pieces string length does not mathematically match the total size and piece length", func(t *testing.T) {
		assert.Panics(t, func() {
			metainfo := serde.MetaInfo{Info: serde.InfoDict{Length: 100, PieceLength: 10, Pieces: strings.Repeat("0", 180)}}
			p.NewTorrentLayout(metainfo, 5)
		})
	})
}

func piecesSample() (ds.LazyArray[protocol.PieceLayout], [][20]byte) {
	blockHash1 := helpers.RandomHash()
	blockHash2 := helpers.RandomHash()
	pieces := ds.NewLazyArrayFromSlice([]p.PieceLayout{
		{
			Hash: blockHash1,
			Size: 3 * 1024,
			Blocks: ds.NewLazyArrayFromSlice([]p.BlockRange{
				{
					Begin: 0,
					Size:  2 * 1024,
				},
				{
					Begin: 2 * 1024,
					Size:  1024,
				},
			}),
		},
		{
			Hash: blockHash2,
			Size: 2 * 1024,
			Blocks: ds.NewLazyArrayFromSlice([]p.BlockRange{
				{
					Begin: 0,
					Size:  2 * 1024,
				},
			}),
		},
	})

	return pieces, [][20]byte{blockHash1, blockHash2}
}

func AssertPieces(t *testing.T, got, want ds.LazyArray[p.PieceLayout]) {
	t.Helper()

	if want.Len() != got.Len() {
		t.Fatalf("piece count mismatch: want %d pieces, got %d", want.Len(), got.Len())
	}

	for i := 0; i < want.Len(); i++ {
		wPiece := want.Get(i)
		gPiece := got.Get(i)

		if wPiece.Hash != gPiece.Hash {
			t.Errorf("piece %d hash mismatch: want %x, got %x", i, wPiece.Hash, gPiece.Hash)
		}
		if wPiece.Size != gPiece.Size {
			t.Errorf("piece %d size mismatch: want %d, got %d", i, wPiece.Size, gPiece.Size)
		}
		if wPiece.Blocks.Len() != gPiece.Blocks.Len() {
			t.Fatalf("piece %d block count mismatch: want %d blocks, got %d", i, wPiece.Blocks.Len(), gPiece.Blocks.Len())
		}

		for j := 0; j < wPiece.Blocks.Len(); j++ {
			wBlock := wPiece.Blocks.Get(j)
			gBlock := gPiece.Blocks.Get(j)

			if wBlock.Begin != gBlock.Begin {
				t.Errorf("piece %d, block %d begin mismatch: want %d, got %d", i, j, wBlock.Begin, gBlock.Begin)
			}
			if wBlock.Size != gBlock.Size {
				t.Errorf("piece %d, block %d size mismatch: want %d, got %d", i, j, wBlock.Size, gBlock.Size)
			}
		}
	}
}
