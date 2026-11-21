package protocol

import (
	"strings"
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/nstefan13/bittorrent/helpers"
	s "github.com/nstefan13/bittorrent/serde"
)

func TestModifyPieceAbundance(t *testing.T) {
	t.Run("panics if pieceIndex is not available", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 5))

		assert.Panics(t, func() {
			pc.ModifyPieceAbundance(100, 0)
		})
	})

	t.Run("increases/decreases abundance only for the specified pieceIndex", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 5))

		pc.ModifyPieceAbundance(0, 23)
		helpers.AssertEqual(t, 23, pc.GetPieceInfo()[0].Abundance)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[1].Abundance)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[2].Abundance)

		pc.ModifyPieceAbundance(0, -2)
		helpers.AssertEqual(t, 21, pc.GetPieceInfo()[0].Abundance)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[1].Abundance)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[2].Abundance)
	})

	t.Run("clamps abundance to a minimum of 0", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 5))

		pc.ModifyPieceAbundance(0, -100)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[0].Abundance)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[1].Abundance)
		helpers.AssertEqual(t, 0, pc.GetPieceInfo()[2].Abundance)
	})
}

func TestSetBlockAsFreeToTake(t *testing.T) {
	t.Run("panics if block is not available", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 5))

		assert.Panics(t, func() {
			pc.SetBlockAsFreeToTake(100, 0)
		})

		assert.Panics(t, func() {
			pc.SetBlockAsFreeToTake(2, 100)
		})
	})

	t.Run("panics if block already is free-to-take", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 5))

		assert.Panics(t, func() {
			pc.SetBlockAsFreeToTake(0, 0)
		})
	})

	t.Run("correctly handles multiple blocks transitioning to FREE_TO_TAKE", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))
		pieceIndex := 0

		// Set all blocks to DOWNLOADING
		pc.setBlockAsDownloading(pieceIndex, 0)
		pc.setBlockAsDownloading(pieceIndex, 1)
		pc.setBlockAsDownloading(pieceIndex, 2)

		helpers.AssertEqual(t, PieceDownloading, pc.GetPieceInfo()[pieceIndex].Status)

		// Free blocks one by one
		pc.SetBlockAsFreeToTake(pieceIndex, 0)
		helpers.AssertEqual(t, PieceDownloading, pc.GetPieceInfo()[pieceIndex].Status) // Still downloading

		pc.SetBlockAsFreeToTake(pieceIndex, 1)
		helpers.AssertEqual(t, PieceDownloading, pc.GetPieceInfo()[pieceIndex].Status) // Still downloading

		// Free last block - should transition to FREE_TO_TAKE
		pc.SetBlockAsFreeToTake(pieceIndex, 2)
		info := pc.GetPieceInfo()[pieceIndex]
		helpers.AssertEqual(t, PieceFreeToTake, info.Status)
		helpers.AssertEqual(t, []BlockStatus(nil), info.BlockStatuses)
	})

	t.Run("transitions piece from WRITTEN to DOWNLOADING when a block is freed", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))
		pieceIndex := 0

		// Drive the piece to WRITTEN state: set all blocks as DOWNLOADING then WRITTEN
		pc.setBlockAsDownloading(pieceIndex, 0)
		pc.setBlockAsDownloading(pieceIndex, 1)
		pc.setBlockAsDownloading(pieceIndex, 2)
		pc.SetBlockAsWritten(pieceIndex, 0)
		pc.SetBlockAsWritten(pieceIndex, 1)
		pc.SetBlockAsWritten(pieceIndex, 2)
		helpers.AssertEqual(t, PieceWritten, pc.GetPieceInfo()[pieceIndex].Status)

		// Free one block — piece should transition back to DOWNLOADING
		pc.SetBlockAsFreeToTake(pieceIndex, 1)
		info := pc.GetPieceInfo()[pieceIndex]
		helpers.AssertEqual(t, PieceDownloading, info.Status)
		// The freed block is FREE_TO_TAKE; the rest are WRITTEN
		helpers.AssertEqual(t, BlockFreeToTake, info.BlockStatuses[1])
		helpers.AssertEqual(t, BlockWritten, info.BlockStatuses[0])
		helpers.AssertEqual(t, BlockWritten, info.BlockStatuses[2])
	})

	t.Run("panics if piece is VALID", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 2))
		pieceIndex := 0

		// Drive the piece all the way to VALID
		pc.setBlockAsDownloading(pieceIndex, 0)
		pc.setBlockAsDownloading(pieceIndex, 1)
		pc.SetBlockAsWritten(pieceIndex, 0)
		pc.SetBlockAsWritten(pieceIndex, 1)
		pc.SetPieceAsValid(pieceIndex)
		helpers.AssertEqual(t, PieceValid, pc.GetPieceInfo()[pieceIndex].Status)

		assert.Panics(t, func() {
			pc.SetBlockAsFreeToTake(pieceIndex, 0)
		})
	})
}

func TestSetBlockAsWritten(t *testing.T) {
	t.Run("panics if block index is out of range", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))
		pc.setBlockAsDownloading(0, 0)

		assert.Panics(t, func() {
			pc.SetBlockAsWritten(0, 100)
		})
	})

	t.Run("panics if piece is not DOWNLOADING", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(2, 3))

		// Piece 0 is FREE_TO_TAKE (initial state)
		assert.Panics(t, func() {
			pc.SetBlockAsWritten(0, 0)
		})
	})

	t.Run("panics if block is not DOWNLOADING", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))
		// Only block 0 is set to DOWNLOADING; block 1 remains FREE_TO_TAKE
		pc.setBlockAsDownloading(0, 0)

		assert.Panics(t, func() {
			pc.SetBlockAsWritten(0, 1)
		})
	})

	t.Run("sets the block to WRITTEN", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))
		pieceIndex := 0

		pc.setBlockAsDownloading(pieceIndex, 0)
		pc.setBlockAsDownloading(pieceIndex, 1)
		pc.SetBlockAsWritten(pieceIndex, 0)

		info := pc.GetPieceInfo()[pieceIndex]
		helpers.AssertEqual(t, PieceDownloading, info.Status)
		helpers.AssertEqual(t, BlockWritten, info.BlockStatuses[0])
		helpers.AssertEqual(t, BlockDownloading, info.BlockStatuses[1])
		helpers.AssertEqual(t, BlockFreeToTake, info.BlockStatuses[2])
	})

	t.Run("transitions piece to WRITTEN when all blocks are WRITTEN", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))
		pieceIndex := 0

		pc.setBlockAsDownloading(pieceIndex, 0)
		pc.setBlockAsDownloading(pieceIndex, 1)
		pc.setBlockAsDownloading(pieceIndex, 2)

		pc.SetBlockAsWritten(pieceIndex, 0)
		helpers.AssertEqual(t, PieceDownloading, pc.GetPieceInfo()[pieceIndex].Status)

		pc.SetBlockAsWritten(pieceIndex, 1)
		helpers.AssertEqual(t, PieceDownloading, pc.GetPieceInfo()[pieceIndex].Status)

		// Writing the last block should transition the piece to WRITTEN
		pc.SetBlockAsWritten(pieceIndex, 2)
		info := pc.GetPieceInfo()[pieceIndex]
		helpers.AssertEqual(t, PieceWritten, info.Status)
		helpers.AssertEqual(t, []BlockStatus(nil), info.BlockStatuses)
	})
}

func TestSetPieceAsValid(t *testing.T) {
	t.Run("panics if piece is not in WRITTEN state", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(2, 3))

		// FREE_TO_TAKE (initial state)
		assert.Panics(t, func() {
			pc.SetPieceAsValid(0)
		})

		// DOWNLOADING state
		pc.setBlockAsDownloading(0, 0)
		assert.Panics(t, func() {
			pc.SetPieceAsValid(0)
		})
	})

	t.Run("sets the piece status to VALID", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 2))
		pieceIndex := 0

		// Drive the piece to WRITTEN
		pc.setBlockAsDownloading(pieceIndex, 0)
		pc.setBlockAsDownloading(pieceIndex, 1)
		pc.SetBlockAsWritten(pieceIndex, 0)
		pc.SetBlockAsWritten(pieceIndex, 1)
		helpers.AssertEqual(t, PieceWritten, pc.GetPieceInfo()[pieceIndex].Status)

		pc.SetPieceAsValid(pieceIndex)
		helpers.AssertEqual(t, PieceValid, pc.GetPieceInfo()[pieceIndex].Status)
	})
}

func TestRequestBlocks(t *testing.T) {
	t.Run("cold start: returns empty when maxNumBlocks is 0", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 5))

		result := pc.RequestBlocks(0)
		helpers.AssertEqual(t, 0, len(result))
	})

	t.Run("cold start: returns empty when no free-to-take pieces remain", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 2))

		// Drive the only piece all the way to VALID
		pc.setBlockAsDownloading(0, 0)
		pc.setBlockAsDownloading(0, 1)
		pc.SetBlockAsWritten(0, 0)
		pc.SetBlockAsWritten(0, 1)
		pc.SetPieceAsValid(0)

		result := pc.RequestBlocks(10)
		helpers.AssertEqual(t, 0, len(result))
	})

	t.Run("cold start: returns all blocks of a piece and marks them as DOWNLOADING", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 3))

		result := pc.RequestBlocks(10)

		helpers.AssertEqual(t, 3, len(result))
		for _, bc := range result {
			helpers.AssertEqual(t, 0, bc.PieceIndex)
		}
		// All blocks should now be DOWNLOADING
		info := pc.GetPieceInfo()[0]
		helpers.AssertEqual(t, PieceDownloading, info.Status)
		for _, s := range info.BlockStatuses {
			helpers.AssertEqual(t, BlockDownloading, s)
		}
	})

	t.Run("cold start: respects maxNumBlocks cap", func(t *testing.T) {
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 5))

		result := pc.RequestBlocks(2)

		helpers.AssertEqual(t, 2, len(result))
		info := pc.GetPieceInfo()[0]
		helpers.AssertEqual(t, PieceDownloading, info.Status)
		numDownloading := 0
		for _, s := range info.BlockStatuses {
			if s == BlockDownloading {
				numDownloading++
			}
		}
		helpers.AssertEqual(t, 2, numDownloading)
	})

	t.Run("warm: prioritizes FREE_TO_TAKE blocks in already-downloading pieces", func(t *testing.T) {
		// 2 pieces, 3 blocks each.
		// Piece 0: partially downloading — triggers the warm path.
		// Piece 1: fully free-to-take.
		// Requesting 2 blocks should fill from piece 0's remaining FREE_TO_TAKE blocks
		// rather than touching piece 1.
		pc := NewPiecePicker(createSimpleTorrentLayout(2, 3))
		pc.setBlockAsDownloading(0, 0)
		// Piece 0: block 0=DOWNLOADING, blocks 1 & 2=FREE_TO_TAKE

		result := pc.RequestBlocks(2)

		helpers.AssertEqual(t, 2, len(result))
		for _, bc := range result {
			helpers.AssertEqual(t, 0, bc.PieceIndex)
		}
		// Piece 1 must remain FREE_TO_TAKE (untouched)
		helpers.AssertEqual(t, PieceFreeToTake, pc.GetPieceInfo()[1].Status)
	})

	t.Run("warm: follows rarest-first policy when picking new free-to-take pieces", func(t *testing.T) {
		// 3 pieces, 2 blocks each.
		// Piece 0: fully DOWNLOADING (no free blocks) — triggers warm path with nothing to fill from downloading.
		// Piece 1: FREE_TO_TAKE, abundance 2 (bucket 2) — common.
		// Piece 2: FREE_TO_TAKE, abundance 1 (bucket 1) — rarer.
		// Requesting 2 blocks should select piece 2 (lowest abundance bucket).
		pc := NewPiecePicker(createSimpleTorrentLayout(3, 2))
		pc.setBlockAsDownloading(0, 0)
		pc.setBlockAsDownloading(0, 1)
		pc.ModifyPieceAbundance(1, 2)
		pc.ModifyPieceAbundance(2, 1)

		result := pc.RequestBlocks(2)

		helpers.AssertEqual(t, 2, len(result))
		for _, bc := range result {
			helpers.AssertEqual(t, 2, bc.PieceIndex)
		}
		// Piece 1 (common) must remain FREE_TO_TAKE
		helpers.AssertEqual(t, PieceFreeToTake, pc.GetPieceInfo()[1].Status)
	})

	t.Run("warm: returned blocks are marked as DOWNLOADING", func(t *testing.T) {
		// Piece 0 partially downloading; requesting the remaining free blocks
		// should mark them DOWNLOADING.
		pc := NewPiecePicker(createSimpleTorrentLayout(1, 4))
		pc.setBlockAsDownloading(0, 0)
		pc.setBlockAsDownloading(0, 1)
		// Blocks 2 & 3 are still FREE_TO_TAKE

		result := pc.RequestBlocks(2)

		helpers.AssertEqual(t, 2, len(result))
		info := pc.GetPieceInfo()[0]
		helpers.AssertEqual(t, BlockDownloading, info.BlockStatuses[result[0].BlockIndex])
		helpers.AssertEqual(t, BlockDownloading, info.BlockStatuses[result[1].BlockIndex])
	})
}

// Creates a TorrentLayoutProvider with `numPieces` pieces, each having `numBlocks` blocks.
// Panics if numPieces or numBlocks are negative or 0
func createSimpleTorrentLayout(numPieces int, numBlocks int) TorrentLayoutProvider {
	if numPieces <= 0 || numBlocks <= 0 {
		panic("createSimpleTorrentLayout: numPieces and numBlocks must be strictly greater than 0")
	}

	metainfo := s.MetaInfo{
		Info: s.InfoDict{
			Length:      1024 * numPieces * numBlocks,
			PieceLength: 1024 * numBlocks,
			Pieces:      strings.Repeat("0", 20*numPieces),
		},
	}

	return NewTorrentLayout(metainfo, 1024)
}
