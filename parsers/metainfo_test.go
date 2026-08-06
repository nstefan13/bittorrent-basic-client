package parsers_test

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"net/url"
	"testing"

	bcd "github.com/nstefan13/bittorrent/bencoding"
	"github.com/nstefan13/bittorrent/helpers"
	"github.com/nstefan13/bittorrent/parsers"
)

func TestParseMetaInfo(t *testing.T) {
	tests := []struct {
		name    string
		torrent []byte
		want    *parsers.MetaInfo
	}{
		{
			name:    "successfully parse a valid torrent in single file mode",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			want: &parsers.MetaInfo{
				TrackerURL: urlFromString("udp://tracker.openbittorrent.com:80"),
				Info: parsers.InfoDict{
					Name:        "test.txt",
					Length:      1024,                   // 1 KB file
					PieceLength: 16384,                  // 16 KB piece size
					Pieces:      "12345678901234567890", // Exactly 1 piece hash (20 bytes)
				},
			},
		},
		{
			name:    "successfully parse a valid torrent in multiple file mode",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024e4:pathl4:dir18:file.txteed6:lengthi2048e4:pathl4:dir29:image.pngeee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			want: &parsers.MetaInfo{
				TrackerURL: urlFromString("udp://tracker.openbittorrent.com:80"),
				Info: parsers.InfoDict{
					Name:        "my_folder",            // Directory name
					PieceLength: 16384,                  // 16 KB piece size
					Pieces:      "12345678901234567890", // Exactly 1 piece hash (20 bytes)
					Files: []parsers.File{
						{
							Length: 1024,
							Path:   []string{"dir1", "file.txt"},
						},
						{
							Length: 2048,
							Path:   []string{"dir2", "image.png"},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsers.ParseMetaInfo(bytes.NewReader(tt.torrent))
			helpers.AssertNoError(t, err)
			helpers.AssertEqual(t, tt.want, got)
		})
	}
}

func TestInfoHash(t *testing.T) {
	t.Run("single file mode", func(t *testing.T) {
		torrent := []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890ee")
		metaInfo, err := parsers.ParseMetaInfo(bytes.NewReader(torrent))
		helpers.AssertNoError(t, err)

		infoDictBytes := []byte("d6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890e")
		expectedHash := sha1.Sum(infoDictBytes)

		helpers.AssertEqual(t, string(expectedHash[:]), metaInfo.Info.InfoHash())
	})

	t.Run("multiple file mode", func(t *testing.T) {
		torrent := []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024e4:pathl4:dir18:file.txteed6:lengthi2048e4:pathl4:dir29:image.pngeee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee")
		metaInfo, err := parsers.ParseMetaInfo(bytes.NewReader(torrent))
		helpers.AssertNoError(t, err)

		infoDictBytes := []byte("d5:filesld6:lengthi1024e4:pathl4:dir18:file.txteed6:lengthi2048e4:pathl4:dir29:image.pngeee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890e")
		expectedHash := sha1.Sum(infoDictBytes)

		helpers.AssertEqual(t, string(expectedHash[:]), metaInfo.Info.InfoHash())
	})
}

func TestParseMetaInfoErrors(t *testing.T) {
	tests := []struct {
		name    string
		torrent []byte
		err     error
	}{
		{
			name:    "key `announce` missing",
			torrent: []byte("d4:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890eee"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `announce` is an invalid URL",
			torrent: []byte("d8:announce6:1http:4:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrMalformedMetaInfo,
		},
		{
			name:    "key `announce` of incorrect type",
			torrent: []byte("d8:announcei1e4:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890eee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `info` missing",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:80e"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `info` of incorrect type",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infoi1ee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `name` missing",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e12:piece lengthi16384e6:pieces20:12345678901234567890eee"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `name` of incorrect type",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:namei1e12:piece lengthi16384e6:pieces20:12345678901234567890eee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `piece length` missing",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:name8:test.txt6:pieces20:12345678901234567890eee"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `piece length` of incorrect type",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:name8:test.txt12:piece length5:163846:pieces20:12345678901234567890eee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `pieces` missing",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384eee"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `pieces` of incorrect type",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:lengthi1024e4:name8:test.txt12:piece lengthi16384e6:piecesi1eee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `length` of incorrect type",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod6:length4:10244:name8:test.txt12:piece lengthi16384e6:pieces20:12345678901234567890eee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `files` of incorrect type",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesi1e4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "incorrect type for `metainfo.files[0]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesli1ee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrMalformedMetaInfo,
		},
		{
			name:    "key `length` missing in `metainfo.files[0]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld4:pathl8:file.txteee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `length` of incorrect type in `metainfo.files[0]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:length4:10244:pathl8:file.txteee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "key `path` missing in `metainfo.files[0]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024eee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrKeyNotFound,
		},
		{
			name:    "key `path` of incorrect type in `metainfo.files[0]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024e4:pathi1eee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrTypeMismatch,
		},
		{
			name:    "incorrect type for `metainfo.files[0].path[0]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024e4:pathli1eeee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrMalformedMetaInfo,
		},
		{
			name:    "key `path` is empty in `metainfo.files[1]`",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024e4:pathl4:dir18:file.txteed6:lengthi2048e4:pathleee4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrMalformedMetaInfo,
		},
		{
			name:    "neither `files` nor `length` is present",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrMalformedMetaInfo,
		},
		{
			name:    "both `files` and `length` are present",
			torrent: []byte("d8:announce35:udp://tracker.openbittorrent.com:804:infod5:filesld6:lengthi1024e4:pathl8:file.txteee6:lengthi1024e4:name9:my_folder12:piece lengthi16384e6:pieces20:12345678901234567890ee"),
			err:     parsers.ErrMalformedMetaInfo,
		},
		{
			name:    "invalid bencode syntax",
			torrent: []byte("invalid bencode"),
			err:     bcd.ErrMalformedInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsers.ParseMetaInfo(bytes.NewReader(tt.torrent))
			helpers.AssertError(t, err, tt.err)
			t.Log("[i]", err.Error())
		})
	}
}

func urlFromString(str string) url.URL {
	result, err := url.Parse(str)
	if err != nil {
		panic(fmt.Sprintf("%q is not a valid URL", str))
	}
	return *result
}
