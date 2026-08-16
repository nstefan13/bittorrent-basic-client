package serde

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/url"

	bcd "github.com/nstefan13/bittorrent/bencoding"
)

type File struct {
	// length
	// The length (in bytes) of the transmitted file
	Length int

	// path
	// "A list of UTF-8 encoded strings corresponding to subdirectory names,
	// the last of which is the actual file name (a zero length list is an error case)."
	Path []string
}

type MetaInfo struct {
	// announce
	// The URL of a tracker
	TrackerURL url.URL

	// info
	Info InfoDict
}

type InfoDict struct {
	// name
	// The suggested name for the file/directory
	Name string

	// piece length
	// A file is divided into multiple pieces, this specifies
	// the maximum length of a piece (only the last piece will
	// have a length smaller than this value)
	PieceLength int

	// pieces
	// A concatenated string of SHA1 hashes for each transmitted piece
	Pieces string

	// --- SINGLE FILE MODE ---
	// length
	// The length (in bytes) of the transmitted file
	Length int

	// --- MULTI FILE MODE ---
	// files
	// A list of structures that describe where each file
	// should be stored
	Files []File
}

// IsMultiFile returns true if the torrent contains multiple files.
func (i *InfoDict) IsMultiFile() bool {
	return len(i.Files) > 0
}

// InfoHash reconstructs the canonical bencoded info dictionary and calculates its 20-byte SHA-1 hash,
// which uniquely identifies the torrent on the BitTorrent network.
func (i *InfoDict) InfoHash() [20]byte {
	buffer := bytes.Buffer{}
	encoder := bcd.NewEncoder(&buffer)
	dict := bcd.BDict{
		"name":         bcd.BString(i.Name),
		"piece length": bcd.BInt(i.PieceLength),
		"pieces":       bcd.BString(i.Pieces),
	}

	if !i.IsMultiFile() {
		dict["length"] = bcd.BInt(i.Length)
	} else {
		files := make(bcd.BList, len(i.Files))
		for index, file := range i.Files {
			path := make(bcd.BList, len(file.Path))
			for index2, sub := range file.Path {
				path[index2] = bcd.BString(sub)
			}
			files[index] = bcd.BDict{
				"length": bcd.BInt(file.Length),
				"path":   path,
			}
		}
		dict["files"] = files
	}

	encoder.EncodeDict(dict)
	hash := sha1.Sum(buffer.Bytes())
	return hash
}

// DeserializeMetaInfo decodes a bencoded dictionary from the provided io.Reader stream
// and maps its fields into a strongly-typed MetaInfo struct.
//
// Error Handling:
//   - If an error occurs during initial bencode decoding (e.g., malformed syntax or
//     unexpected EOF), the underlying bencoding decoder error is wrapped and returned.
//   - If bencode decoding succeeds but the dictionary violates the BitTorrent metainfo
//     specification, a wrapped ErrMalformedMetaInfo is returned.
//
// Validation & Specification Requirements:
//   - The root dictionary must contain a string `announce` (tracker URL) and an `info` dictionary.
//   - The `info` dictionary must contain `name` (string), `piece length` (integer), and `pieces` (string).
//   - Mode exclusivity: The `info` dictionary must contain EITHER `length` (single-file mode)
//     OR `files` (multi-file mode), but never both nor neither.
//   - Multi-file mode: Each element in `files` must be a dictionary with a valid integer `length`
//     and a non-empty list of string path components in `path`.
//
// Returned errors wrap sentinel errors such as ErrMalformedMetaInfo, ErrKeyNotFound, or
// ErrTypeMismatch, allowing callers to inspect specific failure conditions using errors.Is.
func DeserializeMetaInfo(reader io.Reader) (metaInfo *MetaInfo, err error) {
	metaInfo = &MetaInfo{}
	decoder := bcd.NewDecoder(reader)
	dict, err := decoder.DecodeDict()
	if err != nil {
		err = fmt.Errorf("error while decoding the metainfo dictionary: %w", err)
		return
	}

	announce, err := ExtractValueFromBDict[bcd.BString](dict, "announce")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	announceAsURL, err := url.Parse(string(announce))
	if err != nil || announceAsURL.Scheme == "" || announceAsURL.Host == "" {
		err = fmt.Errorf("%w: `announce` is an invalid URL", ErrMalformedMetaInfo)
		return
	}
	metaInfo.TrackerURL = *announceAsURL

	info, err := ExtractValueFromBDict[bcd.BDict](dict, "info")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}

	name, err := ExtractValueFromBDict[bcd.BString](info, "name")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	metaInfo.Info.Name = string(name)

	pieceLength, err := ExtractValueFromBDict[bcd.BInt](info, "piece length")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	metaInfo.Info.PieceLength = int(pieceLength)

	pieces, err := ExtractValueFromBDict[bcd.BString](info, "pieces")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	metaInfo.Info.Pieces = string(pieces)

	// We need to check something:
	//	that we have EITHER a `length` or a `files`
	_, lengthExists := info["length"]
	_, filesExists := info["files"]

	if !lengthExists && !filesExists {
		err = fmt.Errorf("%w: %w: neither a `length` nor a `files` key exists", ErrMalformedMetaInfo, ErrKeyNotFound)
		return
	}
	if lengthExists && filesExists {
		err = fmt.Errorf("%w: both `length` and `files` keys exist", ErrMalformedMetaInfo)
		return
	}

	if lengthExists {
		length, err := ExtractValueFromBDict[bcd.BInt](info, "length")
		if err != nil {
			return nil, &MalformedMetaInfoError{err}
		}
		metaInfo.Info.Length = int(length)
	}

	if filesExists {
		files, err2 := ExtractValueFromBDict[bcd.BList](info, "files")
		if err2 != nil {
			return nil, &MalformedMetaInfoError{err2}
		}
		metaInfo.Info.Files = make([]File, len(files))

		for index, untypedFile := range files {
			file, ok := untypedFile.(bcd.BDict)
			if !ok {
				return nil, fmt.Errorf("%w: the %d-th entry in `metainfo.files` is not a dictionary", ErrMalformedMetaInfo, index)
			}

			length, err2 := ExtractValueFromBDict[bcd.BInt](file, "length")
			if err2 != nil {
				return nil, fmt.Errorf("%w: for the %d-th entry in `metainfo.files`: %w", ErrMalformedMetaInfo, index, err2)
			}
			metaInfo.Info.Files[index].Length = int(length)

			path, err2 := ExtractValueFromBDict[bcd.BList](file, "path")
			if err2 != nil {
				return nil, fmt.Errorf("%w: for the %d-th entry in `metainfo.files`: %w", ErrMalformedMetaInfo, index, err2)
			}
			if len(path) == 0 {
				return nil, fmt.Errorf("%w: `metainfo.files[%d].path` is empty", ErrMalformedMetaInfo, index)
			}
			metaInfo.Info.Files[index].Path = make([]string, len(path))

			for index2, untypedSubdirectory := range path {
				subdirectory, ok := untypedSubdirectory.(bcd.BString)
				if !ok {
					return nil, fmt.Errorf("%w: the %d-th item of `metainfo.files[%d].path` is not a string", ErrMalformedMetaInfo, index2, index)
				}

				metaInfo.Info.Files[index].Path[index2] = string(subdirectory)
			}
		}
	}

	return
}
