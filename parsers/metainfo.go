package parsers

import (
	"errors"
	"fmt"
	"io"
	"reflect"

	bcd "github.com/nstefan13/bittorrent/bencoding"
)

var ErrMalformedMetaInfo = errors.New("torrent is corrupt")
var ErrKeyNotFound = errors.New("key does not exists")
var ErrTypeMismatch = errors.New("value has incorrect type")

type MalformedMetaInfoError struct {
	Err error
}

func (e *MalformedMetaInfoError) Error() string {
	return fmt.Sprintf("%v: %v", ErrMalformedMetaInfo, e.Err)
}

// Unwrap returns both the sentinel error and the underlying error
// so errors.Is(err, ErrMalformedMetaInfo) and errors.Is(err, e.Err) both return true.
func (e *MalformedMetaInfoError) Unwrap() []error {
	return []error{ErrMalformedMetaInfo, e.Err}
}

type KeyNotFoundError struct {
	Key string
}

func (e *KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %q not found", e.Key)
}

func (e *KeyNotFoundError) Unwrap() error {
	return ErrKeyNotFound
}

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
	TrackerURL string

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

// Decodes the metainfo dictionary encoded in the provided
// stream and maps it to a MetaInfo struct
func ParseMetaInfo(reader io.Reader) (metaInfo *MetaInfo, err error) {
	metaInfo = &MetaInfo{}
	decoder := bcd.NewDecoder(reader)
	dict, err := decoder.DecodeDict()
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}

	announce, err := extractValue[bcd.BString](dict, "announce")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	metaInfo.TrackerURL = string(announce)

	info, err := extractValue[bcd.BDict](dict, "info")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}

	name, err := extractValue[bcd.BString](info, "name")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	metaInfo.Info.Name = string(name)

	pieceLength, err := extractValue[bcd.BInt](info, "piece length")
	if err != nil {
		err = &MalformedMetaInfoError{err}
		return
	}
	metaInfo.Info.PieceLength = int(pieceLength)

	pieces, err := extractValue[bcd.BString](info, "pieces")
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
		length, err := extractValue[bcd.BInt](info, "length")
		if err != nil {
			return nil, &MalformedMetaInfoError{err}
		}
		metaInfo.Info.Length = int(length)
	}

	if filesExists {
		files, err2 := extractValue[bcd.BList](info, "files")
		if err2 != nil {
			return nil, &MalformedMetaInfoError{err2}
		}
		metaInfo.Info.Files = make([]File, len(files))

		for index, untypedFile := range files {
			file, ok := untypedFile.(bcd.BDict)
			if !ok {
				return nil, fmt.Errorf("%w: the %d-th entry in `metainfo.files` is not a dictionary", ErrMalformedMetaInfo, index)
			}

			length, err2 := extractValue[bcd.BInt](file, "length")
			if err2 != nil {
				return nil, fmt.Errorf("%w: for the %d-th entry in `metainfo.files`: %w", ErrMalformedMetaInfo, index, err2)
			}
			metaInfo.Info.Files[index].Length = int(length)

			path, err2 := extractValue[bcd.BList](file, "path")
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

func extractValue[ValueType any](dict bcd.BDict, key string) (value ValueType, err error) {
	untypedValue, alreadyThere := dict[key]
	if !alreadyThere {
		err = &KeyNotFoundError{key}
		return
	}

	value, ok := dict[key].(ValueType)
	if !ok {
		err = fmt.Errorf("%w: value corresponding to key %q should be of type %q, not %q", ErrTypeMismatch, key, reflect.TypeFor[ValueType]().Name(), reflect.TypeOf(untypedValue).Name())
		return
	}

	return
}
