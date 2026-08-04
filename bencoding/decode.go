package bencoding

import (
	"bufio"
	"io"
	"strconv"
)

func DecodeString(reader *bufio.Reader) (data []byte) {
	prefix, _ := reader.ReadString(':')
	prefix = prefix[:len(prefix)-1] // potential runtime error
	length, _ := strconv.Atoi(prefix)

	data = make([]byte, length)
	io.ReadFull(reader, data)

	return
}
