///////// CONSTANTS /////////

package protocol

import (
	"fmt"
	"math/rand/v2"

	"github.com/emirpasic/gods/v2/sets/hashset"
	"github.com/emirpasic/gods/v2/sets/linkedhashset"
)

type BlockStatus string
type PieceStatus string

const (
	BlockFreeToTake  BlockStatus = "FreeToTake"
	BlockDownloading BlockStatus = "Downloading"
	BlockWritten     BlockStatus = "Valid"
)

const (
	PieceFreeToTake  PieceStatus = "FreeToTake"
	PieceDownloading PieceStatus = "Downloading"
	PieceWritten     PieceStatus = "Written"
	PieceValid       PieceStatus = "Valid"
)

const NUM_ABUNDANCE_BUCKETS = 20

////// AUXILIARY STRUCTS //////

// Used to uniquely identify a block. It simply stores
// the piece index of the piece it belongs to and its
// index inside that piece
type BlockCoordinates struct {
	PieceIndex int
	BlockIndex int
}

type PieceInfo struct {
	Status    PieceStatus
	Abundance int

	// non-nil ONLY when status is PieceDownloading
	// used to track the progress of each block
	BlockStatuses []BlockStatus
}

/////// THE MAIN STRUCT ///////

type PiecePicker struct {
	// To know how many pieces are there and how many blocks does each piece have
	layout    TorrentLayoutProvider
	pieceInfo []PieceInfo

	// store indicies of pieces that are free-to-take/downloading
	downloadingPieces *linkedhashset.Set[int]
	freeToTakePieces  []*hashset.Set[int]
}

// NewPiecePicker creates a PiecePicker for the given torrent layout.
// All pieces start in PieceFreeToTake state with an abundance of 0, placed in
// abundance bucket 0. The caller is expected to call ModifyPieceAbundance as
// peer bitfields are received so that the rarest-first policy has accurate data.
func NewPiecePicker(layout TorrentLayoutProvider) *PiecePicker {
	pieceCount := layout.Pieces().Len()

	pc := &PiecePicker{}
	pc.layout = layout

	pc.pieceInfo = make([]PieceInfo, pieceCount)
	for i := range pc.pieceInfo {
		pc.pieceInfo[i].Status = PieceFreeToTake
	}

	pc.freeToTakePieces = make([]*hashset.Set[int], NUM_ABUNDANCE_BUCKETS)
	for i := range NUM_ABUNDANCE_BUCKETS {
		pc.freeToTakePieces[i] = hashset.New[int]()
	}
	for i := range pieceCount {
		pc.freeToTakePieces[0].Add(i)
	}

	pc.downloadingPieces = linkedhashset.New[int]()

	return pc
}

// RequestBlocks returns up to maxNumBlocks block coordinates that are ready to
// be requested from a peer, and immediately marks all of them as DOWNLOADING.
//
// Expectations:
//   - maxNumBlocks >= 0. Passing 0 is valid and returns an empty slice.
//
// Guarantees:
//   - The returned slice never exceeds maxNumBlocks in length.
//   - Every returned block was FREE_TO_TAKE before this call; after the call it
//     is DOWNLOADING.
//   - Blocks are never returned twice (i.e. a block already DOWNLOADING is
//     never included in the result).
//
// How it operates — two distinct modes:
//
//  1. Cold-start (no pieces currently downloading):
//     Picks a single free-to-take piece at random and returns up to maxNumBlocks
//     of its blocks in order. Only one piece is started so that the caller can
//     quickly pivot if the peer turns out not to have it.
//
//  2. Warm path (at least one piece already downloading):
//     First fills capacity from FREE_TO_TAKE blocks inside already-downloading
//     pieces (in the order the pieces were started). Once those are exhausted, or
//     if capacity remains, it applies a rarest-first policy over the free-to-take
//     pieces: iterates abundance buckets from lowest to highest and picks entire
//     pieces until maxNumBlocks is reached.
func (pc *PiecePicker) RequestBlocks(maxNumBlocks int) []BlockCoordinates {
	res := make([]BlockCoordinates, 0, maxNumBlocks)
	leftToCollect := maxNumBlocks

	// random piece selection when there are no downloadingPieces (I know it should be only at the beginning tho)
	if pc.downloadingPieces.Size() == 0 {
		if leftToCollect > 0 && pc.numFreeToTakePieces() > 0 {

			pieceIndex := pc.randomFreeToTakePieceIndex()
			blocksAvailable := pc.layout.Pieces().Get(pieceIndex).Blocks.Len()

			for blockIndex := range blocksAvailable {
				if leftToCollect > 0 {
					res = append(res, BlockCoordinates{pieceIndex, blockIndex})
					leftToCollect--
				}
			}
		}

		for _, bc := range res {
			pc.setBlockAsDownloading(bc.PieceIndex, bc.BlockIndex)
		}
		return res
	}

	// prioritise downloading blocks
	for it := pc.downloadingPieces.Iterator(); it.Next(); {
		pi := &pc.pieceInfo[it.Value()]

		for blockIndex, s := range pi.BlockStatuses {
			if s == BlockFreeToTake {
				if leftToCollect > 0 {
					res = append(res, BlockCoordinates{it.Value(), blockIndex})
					leftToCollect--
				}
			}
		}
	}

	// rarest first policy
outerloop:
	for _, bucket := range pc.freeToTakePieces {
		for _, pieceIndex := range bucket.Values() {
			if leftToCollect == 0 {
				break
			}

			blocksAvailable := pc.layout.Pieces().Get(pieceIndex).Blocks.Len()
			for blockIndex := range blocksAvailable {
				if leftToCollect > 0 {
					res = append(res, BlockCoordinates{pieceIndex, blockIndex})
					leftToCollect--
				} else {
					break outerloop
				}
			}
		}
	}

	// set all the block we've collected to DOWNLOADING and return them
	for _, bc := range res {
		pc.setBlockAsDownloading(bc.PieceIndex, bc.BlockIndex)
	}
	return res
}

// ModifyPieceAbundance adjusts the abundance (peer availability) of a piece.
// If the piece is in PieceFreeToTake state, it moves the piece to the appropriate
// abundance bucket. The abundance is clamped to a minimum of 0.
func (pc *PiecePicker) ModifyPieceAbundance(pieceIndex int, delta int) {
	oldAbundance := pc.pieceInfo[pieceIndex].Abundance
	newAbundance := max(0, oldAbundance+delta)

	pc.pieceInfo[pieceIndex].Abundance = newAbundance

	// this piece is in `freeToTake` and its abundance has
	// just been modified, we need to move it to another bucket
	if pc.pieceInfo[pieceIndex].Status == PieceFreeToTake {
		oldBucket := min(oldAbundance, NUM_ABUNDANCE_BUCKETS-1)
		newBucket := min(newAbundance, NUM_ABUNDANCE_BUCKETS-1)

		pc.freeToTakePieces[oldBucket].Remove(pieceIndex)
		pc.freeToTakePieces[newBucket].Add(pieceIndex)
	}
}

// SetBlockAsFreeToTake marks a block as free to take.
// Expectations:
// - For PieceFreeToTake pieces: block must be FREE_TO_TAKE initially (panic otherwise)
// - For PieceDownloading pieces: block can be DOWNLOADING or WRITTEN
// - For PieceWritten pieces: transitions the piece to DOWNLOADING state
// - PieceValid pieces: panic (no operations allowed)
// Behavior: When the last DOWNLOADING block is freed, the piece transitions back to
// PieceFreeToTake and is re-added to the appropriate abundance bucket.
func (pc *PiecePicker) SetBlockAsFreeToTake(pieceIndex int, blockIndex int) {
	pi := &pc.pieceInfo[pieceIndex]
	numBlocks := pc.layout.Pieces().Get(pieceIndex).Blocks.Len()
	if pi.Status == PieceFreeToTake ||
		(pi.Status == PieceDownloading && pi.BlockStatuses[blockIndex] == BlockFreeToTake) {
		panic(fmt.Sprintf("SetBlockAsFreeToTake(pieceIndex=%d, blockIndex=%d): block already is FREE_TO_TAKE", pieceIndex, blockIndex))
	}
	if pi.Status == PieceValid {
		panic(fmt.Sprintf("SetBlockAsFreeToTake(pieceIndex=%d, blockIndex=%d): piece is VALID, no operations can be performed on it", pieceIndex, blockIndex))
	}
	if blockIndex < 0 || blockIndex >= numBlocks {
		panic(fmt.Sprintf("SetBlockAsFreeToTake(pieceIndex=%d, blockIndex=%d): block index out of range (numBlocks=%d)", pieceIndex, blockIndex, numBlocks))
	}

	if pi.Status == PieceDownloading {
		// set the block to FREE_TO_TAKE
		pi.BlockStatuses[blockIndex] = BlockFreeToTake

		// if all other blocks are also FREE_TO_TAKE:
		// 	- set the piece to FREE_TO_TAKE
		//	- remove `BlockStatuses`
		//  - remove it from `downloadingPieces`
		//  - add it to `freeToTakePieces`
		numTotalBlocks := len(pi.BlockStatuses)
		numFreeToTakeBlocks := 0
		for _, s := range pi.BlockStatuses {
			if s == BlockFreeToTake {
				numFreeToTakeBlocks++
			}
		}

		if numFreeToTakeBlocks == numTotalBlocks {
			pi.Status = PieceFreeToTake
			pi.BlockStatuses = nil
			pc.downloadingPieces.Remove(pieceIndex)
			// re-add the piece to `freeToTakePieces` at its abundance bucket
			bucketIndex := min(pi.Abundance, NUM_ABUNDANCE_BUCKETS-1)
			pc.freeToTakePieces[bucketIndex].Add(pieceIndex)
		}

		return
	}

	if pi.Status == PieceWritten {
		// set the piece to DOWNLOADING
		pi.Status = PieceDownloading

		// populate BlockStatuses with all blocks initially WRITTEN
		pi.BlockStatuses = initBlockStatuses(numBlocks, BlockWritten)
		pi.BlockStatuses[blockIndex] = BlockFreeToTake

		// add the piece to `downloadingPieces`
		pc.downloadingPieces.Add(pieceIndex)

		return
	}
}

// SetBlockAsWritten marks a block as written (successfully downloaded and verified).
// Expectations:
// - Piece must be in PieceDownloading state
// - Block must be in BlockDownloading state
// Panics if these conditions are not met.
// Behavior: When all blocks are written, the piece transitions to PieceWritten state
// and is removed from downloadingPieces. BlockStatuses is cleared.
func (pc *PiecePicker) SetBlockAsWritten(pieceIndex int, blockIndex int) {
	pi := &pc.pieceInfo[pieceIndex]
	numBlocks := pc.layout.Pieces().Get(pieceIndex).Blocks.Len()
	if blockIndex < 0 || blockIndex >= numBlocks {
		panic(fmt.Sprintf("SetBlockAsWritten(pieceIndex=%d, blockIndex=%d): block index out of range (numBlocks=%d)", pieceIndex, blockIndex, numBlocks))
	}
	if pi.Status != PieceDownloading {
		panic(fmt.Sprintf("SetBlockAsWritten(pieceIndex=%d, blockIndex=%d): piece must be DOWNLOADING (got %q)", pieceIndex, blockIndex, pi.Status))
	}
	if pi.BlockStatuses[blockIndex] != BlockDownloading {
		panic(fmt.Sprintf("SetBlockAsWritten(pieceIndex=%d, blockIndex=%d): block must be DOWNLOADING (got %q)", pieceIndex, blockIndex, pi.BlockStatuses[blockIndex]))
	}

	// set the block to WRITTEN
	pi.BlockStatuses[blockIndex] = BlockWritten

	// if all blocks are WRITTEN
	// 	- change state of piece to WRITTEN
	//	- remove from `downloadingPieces`
	//	- set BlockStatuses to nil
	numWrittenBlocks := 0
	for _, s := range pi.BlockStatuses {
		if s == BlockWritten {
			numWrittenBlocks++
		}
	}

	if numBlocks == numWrittenBlocks {
		pi.Status = PieceWritten
		pc.downloadingPieces.Remove(pieceIndex)
		pi.BlockStatuses = nil
	}
}

// Sets the status of a piece to VALID
// Expectations (panics if not met):
// - Piece must be in PieceWritten state
func (pc *PiecePicker) SetPieceAsValid(pieceIndex int) {
	pi := &pc.pieceInfo[pieceIndex]
	if pi.Status != PieceWritten {
		panic(fmt.Sprintf("SetPieceAsValid(pieceIndex=%d): piece is in state %q, instead of WRITTEN", pieceIndex, pi.Status))
	}

	pi.Status = PieceValid
}

// GetPieceInfo returns a copy of all piece information.
// The returned slice is a copy, so modifications do not affect the internal state.
func (pc *PiecePicker) GetPieceInfo() []PieceInfo {
	return append([]PieceInfo(nil), pc.pieceInfo...)
}

// It expects the piece to be in FREE_TO_TAKE or DOWNLOADING and the specified block
// to be in DOWNLOADING. If these expectations are not met, it panics.
func (pc *PiecePicker) setBlockAsDownloading(pieceIndex int, blockIndex int) {
	pi := &pc.pieceInfo[pieceIndex]
	numBlocks := pc.layout.Pieces().Get(pieceIndex).Blocks.Len()
	if !(pi.Status == PieceFreeToTake || pi.Status == PieceDownloading) {
		panic(fmt.Sprintf("setBlockAsDownloading(pieceIndex=%d, blockIndex=%d): invalid piece status %q (expected FREE_TO_TAKE or DOWNLOADING)", pieceIndex, blockIndex, pi.Status))
	}
	if blockIndex < 0 || blockIndex >= numBlocks {
		panic(fmt.Sprintf("setBlockAsDownloading(pieceIndex=%d, blockIndex=%d): block index out of range (numBlocks=%d)", pieceIndex, blockIndex, numBlocks))
	}

	if pi.Status == PieceFreeToTake {
		// remove it from `freeToTakePieces`
		bucketIndex := min(pi.Abundance, NUM_ABUNDANCE_BUCKETS-1)
		pc.freeToTakePieces[bucketIndex].Remove(pieceIndex)

		// set status to DOWNLOADING
		pi.Status = PieceDownloading

		// populate BlockStatuses with all blocks initially FREE_TO_TAKE
		pi.BlockStatuses = initBlockStatuses(numBlocks, BlockFreeToTake)
		pi.BlockStatuses[blockIndex] = BlockDownloading

		// add it to `downloadingPieces`
		pc.downloadingPieces.Add(pieceIndex)

		return
	}

	if pi.Status == PieceDownloading {
		if pi.BlockStatuses[blockIndex] != BlockFreeToTake {
			panic(fmt.Sprintf("setBlockAsDownloading(pieceIndex=%d, blockIndex=%d): block already downloading (status=%q)", pieceIndex, blockIndex, pi.BlockStatuses[blockIndex]))
		}

		pi.BlockStatuses[blockIndex] = BlockDownloading

		return
	}
}

// initBlockStatuses allocates and initializes a BlockStatuses array with the given default value.
func initBlockStatuses(numBlocks int, defaultValue BlockStatus) []BlockStatus {
	blockStatuses := make([]BlockStatus, numBlocks)
	for i := range numBlocks {
		blockStatuses[i] = defaultValue
	}
	return blockStatuses
}

// numFreeToTakePieces returns the total number of pieces currently in
// PieceFreeToTake state by summing the sizes of all abundance buckets.
func (pc *PiecePicker) numFreeToTakePieces() int {
	var res int

	for _, bucket := range pc.freeToTakePieces {
		res += bucket.Size()
	}

	return res
}

// randomFreeToTakePieceIndex returns the index of a randomly selected
// free-to-take piece. The selection is bucket-uniform, not piece-uniform:
// it picks a random non-empty abundance bucket first, then a random piece
// within that bucket. This means pieces in smaller buckets have a higher
// per-piece probability of being chosen.
// Panics if there are no free-to-take pieces.
func (pc *PiecePicker) randomFreeToTakePieceIndex() (pieceIndex int) {
	var bucketsContainingFreeToTakePieces []int
	for index, bucket := range pc.freeToTakePieces {
		if bucket.Size() > 0 {
			bucketsContainingFreeToTakePieces = append(bucketsContainingFreeToTakePieces, index)
		}
	}

	randomBucketIndex := bucketsContainingFreeToTakePieces[rand.IntN(len(bucketsContainingFreeToTakePieces))]
	randomElementIndex := rand.IntN(pc.freeToTakePieces[randomBucketIndex].Size())

	pieceIndex = pc.freeToTakePieces[randomBucketIndex].Values()[randomElementIndex]
	return
}
