package protocol

import (
	"fmt"

	ds "github.com/nstefan13/bittorrent/datastructures"
	"github.com/nstefan13/bittorrent/serde"
)

type BlockInfo struct {
	Begin int
	Size  int
}

type PieceInfo struct {
	Hash   [20]byte
	Size   int
	Blocks ds.LazyArray[BlockInfo]
}

type TorrentLayoutProvider interface {
	// Pieces returns a lazy array of all pieces in the torrent.
	// For each piece it MUST enforce that it has at least 1 block,
	// if it cannot enforce this requirement, it should PANIC.
	Pieces() ds.LazyArray[PieceInfo]
}

///// ACTUAL IMPLEMENTATIONS

type TorrentLayout struct {
	torrentSize int
	pieceLength int
	blockLength int
	hashes      string
}

// Constructs a TorrentLayoutProvider from a MetaInfo dictionary with the size of the
// blocks being at most `blockSize`. If `blockSize` is greater than the size of a piece,
// it is automatically capped to the size of a piece (i.e. `blockSize = pieceSize`).
//
// Panics when the provided metainfo is invalid as it assumes the user will always provide a valid
// metainfo (usually serialized and validated by [serde.DeserializeMetaInfo])
func NewTorrentLayout(metainfo serde.MetaInfo, blockSize int) TorrentLayoutProvider {
	var totalSize int
	if metainfo.Info.IsMultiFile() {
		for _, file := range metainfo.Info.Files {
			if file.Length <= 0 {
				panic(fmt.Sprintf("invalid file length: %d", file.Length))
			}
			totalSize += file.Length
		}
	} else {
		totalSize = metainfo.Info.Length
	}

	if blockSize > metainfo.Info.PieceLength {
		blockSize = metainfo.Info.PieceLength
	}

	if totalSize <= 0 {
		panic(fmt.Sprintf("invalid total torrent size: %d", totalSize))
	}
	if metainfo.Info.PieceLength <= 0 {
		panic(fmt.Sprintf("invalid piece length: %d", metainfo.Info.PieceLength))
	}
	if blockSize <= 0 {
		panic(fmt.Sprintf("invalid block size: %d", blockSize))
	}
	if len(metainfo.Info.Pieces)%20 != 0 {
		panic(fmt.Sprintf("pieces string length (%d) is not a multiple of 20", len(metainfo.Info.Pieces)))
	}

	expectedPieces := (totalSize + metainfo.Info.PieceLength - 1) / metainfo.Info.PieceLength
	actualPieces := len(metainfo.Info.Pieces) / 20

	if expectedPieces != actualPieces {
		panic(fmt.Sprintf("pieces string length mismatch: expected %d pieces based on size, got %d", expectedPieces, actualPieces))
	}

	return TorrentLayout{
		torrentSize: totalSize,
		pieceLength: metainfo.Info.PieceLength,
		blockLength: blockSize,
		hashes:      metainfo.Info.Pieces,
	}
}

// Pieces returns a lazy array of all pieces in the torrent, generating piece sizes,
// hashes, and block breakdowns on the fly without allocating large arrays in memory.
func (l TorrentLayout) Pieces() ds.LazyArray[PieceInfo] {
	numPieces := len(l.hashes) / 20
	return ds.NewLazyArray(numPieces, func(pieceIndex int) PieceInfo {
		pieceSize := min(l.pieceLength, l.torrentSize-pieceIndex*l.pieceLength)
		if pieceSize <= 0 || l.blockLength <= 0 {
			panic(fmt.Sprintf("invalid piece/block size: piece=%d, block=%d", pieceSize, l.blockLength))
		}

		var hash [20]byte
		copy(hash[:], l.hashes[pieceIndex*20:(pieceIndex+1)*20])

		numBlocks := 1 + (pieceSize-1)/l.blockLength
		if numBlocks < 1 {
			panic("a piece must have at least one block")
		}

		return PieceInfo{
			Hash: hash,
			Size: pieceSize,
			Blocks: ds.NewLazyArray(numBlocks, func(blockIndex int) BlockInfo {
				blockBegin := blockIndex * l.blockLength
				blockSize := min(l.blockLength, pieceSize-blockIndex*l.blockLength)
				return BlockInfo{
					Begin: blockBegin,
					Size:  blockSize,
				}
			}),
		}
	})
}
