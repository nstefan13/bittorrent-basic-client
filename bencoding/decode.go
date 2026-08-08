package bencoding

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// BStringMaxLength is the maximum allowed byte length for a bencoded string (64 MB).
// This hard limit prevents attackers from triggering massive memory allocations with crafted length prefixes.
const BStringMaxLength = 64 * 1024 * 1024

// Decoder parses bencoded data streams into intermediate bencoding representations
// (BInt, BString, BList, BDict).
type Decoder struct {
	_trackedReader *TrackedReader // only Offset() should use it
	reader         *bufio.Reader
}

// NewDecoder creates a new bencoding [Decoder] reading from the supplied reader.
func NewDecoder(reader io.Reader) (decoder *Decoder) {
	decoder = &Decoder{}
	decoder._trackedReader = &TrackedReader{reader: reader}
	decoder.reader = bufio.NewReader(decoder._trackedReader)
	return decoder
}

// Offset returns the current 0-indexed byte offset in the underlying stream.
// Useful for creating detailed, actionable parsing error messages.
func (d *Decoder) Offset() int {
	return d._trackedReader.offset - d.reader.Buffered()
}

// DecodeString decodes a bencoded byte string ("<length>:<contents>").
//
// The length prefix must be a non-negative base-10 integer followed by a colon.
// To prevent denial-of-service attacks via unbounded memory allocation, string lengths
// are capped at [BStringMaxLength] (64 MB).
//
// Error conditions:
//   - Returns [ErrInvalidStringLength] if the length prefix is missing, non-numeric, negative, or exceeds [BStringMaxLength].
//   - Returns [io.ErrUnexpectedEOF] if the stream ends prematurely before reading the full string.
func (d *Decoder) DecodeString() (data BString, err error) {
	startingOffset := d.Offset()
	prefix, err := d.reader.ReadString(':')
	if err = treatEOFAsUnexpected(err); err != nil {
		err = fmt.Errorf("parsing BString starting at offset %d: %w", startingOffset, err)
		return
	}

	prefix = prefix[:len(prefix)-1]
	length, err := strconv.Atoi(prefix)
	if err != nil {
		err = fmt.Errorf("parsing BString starting at offset %d: %w: %w", startingOffset, ErrInvalidStringLength, err)
		return
	}
	if length < 0 {
		err = fmt.Errorf("parsing BString starting at offset %d: %w: length cannot be negative", startingOffset, ErrInvalidStringLength)
		return
	}
	if length >= BStringMaxLength {
		err = fmt.Errorf("parsing BString starting at offset %d: %w: length larger than the allowed limit of %d bytes", startingOffset, ErrInvalidStringLength, BStringMaxLength)
		return
	}

	buffer := make([]byte, length)
	_, err = io.ReadFull(d.reader, buffer)
	data = BString(buffer)
	if err = treatEOFAsUnexpected(err); err != nil {
		err = fmt.Errorf("parsing BString starting at offset %d: %w", startingOffset, err)
		return
	}

	return
}

// DecodeInt decodes a bencoded integer ("i<integer>e") into a [BInt].
//
// Per the BEP3 specification:
//   - Integers must be prefixed with 'i' and terminated with 'e'.
//   - Leading zeros are not allowed (e.g. "i03e" is invalid), except for "i0e".
//   - Negative zero ("i-0e") is invalid.
//
// Error conditions:
//   - Returns [ErrMalformedInput] if the stream does not begin with 'i'.
//   - Returns [ErrInvalidIntegerValue] if the value is empty, non-numeric, contains invalid signs or leading zeros.
//   - Returns [io.ErrUnexpectedEOF] if the stream ends prematurely before reading the terminating 'e'.
func (d *Decoder) DecodeInt() (value BInt, err error) {
	startingOffset := d.Offset()
	got, err := d.reader.ReadByte()
	if err = treatEOFAsUnexpected(err); err != nil {
		err = fmt.Errorf("parsing BInt starting at offset %d: %w", startingOffset, err)
		return
	}
	if got != 'i' {
		err = fmt.Errorf("parsing BInt starting at offset %d: %w: a bencoded integer must start with byte `i`", startingOffset, ErrMalformedInput)
		return
	}

	data, err := d.reader.ReadString('e')
	if err = treatEOFAsUnexpected(err); err != nil {
		err = fmt.Errorf("parsing BInt starting at offset %d: %w", startingOffset, err)
		return
	}

	data = data[:len(data)-1]
	valueAsInt, err := strconv.Atoi(data)
	if err != nil {
		err = fmt.Errorf("parsing BInt starting at offset %d: %w: %w", startingOffset, ErrInvalidIntegerValue, err)
		return
	}

	// by now, we are sure that the raw value was a valid integer
	// due to the use of Atoi, but there is still the possibility
	// of the input to contain leading zeros or the negative zero
	// (-0 or -00). Fortunately, this is quite easy to verify
	valueAsString := strconv.Itoa(valueAsInt)
	if valueAsString != data {
		err = fmt.Errorf("parsing BInt starting at offset %d: %w (data=%q)", startingOffset, ErrInvalidIntegerValue, data)
		return
	}

	value = BInt(valueAsInt)
	return
}

// DecodeList decodes a bencoded list of elements ("l<elements>e") into a [BList].
//
// Elements are parsed recursively using [Decoder.DecodeAny].
//
// Error conditions:
//   - Returns [ErrMalformedInput] if the list does not start with 'l'.
//   - Returns any error encountered while parsing elements within the list.
//   - Returns [io.ErrUnexpectedEOF] if the stream terminates before finding the closing 'e'.
func (d *Decoder) DecodeList() (list BList, err error) {
	startingOffset := d.Offset()
	list = make([]any, 0)
	got, err := d.reader.ReadByte()
	if err = treatEOFAsUnexpected(err); err != nil {
		err = fmt.Errorf("parsing BList starting at offset %d: %w", startingOffset, err)
		return
	}
	if got != 'l' {
		err = fmt.Errorf("parsing BList starting at offset %d: %w: a bencoded list must start with byte `l`", startingOffset, ErrMalformedInput)
		return
	}

	for {
		yes, err := areWeFinished(d.reader)
		if err = treatEOFAsUnexpected(err); err != nil {
			return nil, fmt.Errorf("parsing BList starting at offset %d (offset after error triggered: %d): %w", startingOffset, d.Offset(), err)
		}
		if yes {
			break
		}

		element, err := d.DecodeAny()
		if err != nil {
			return nil, err
		}

		list = append(list, element)
	}

	// This will read the last 'e' (and we can
	// surely read it because previously we peeked
	// at it)
	d.reader.ReadByte()

	return
}

// DecodeDict decodes a bencoded dictionary ("d<key><val>...e") into a [BDict].
//
// In BEP3, all dictionary keys must be bencoded strings. Values are decoded recursively
// via [Decoder.DecodeAny].
//
// Error conditions:
//   - Returns [ErrMalformedInput] if the dictionary does not begin with 'd'.
//   - Returns parsing errors encountered while decoding keys or values.
//   - Returns [io.ErrUnexpectedEOF] if the stream ends before reading the terminating 'e'.
func (d *Decoder) DecodeDict() (dict BDict, err error) {
	startingOffset := d.Offset()
	dict = make(map[string]any)

	got, err := d.reader.ReadByte()
	if err = treatEOFAsUnexpected(err); err != nil {
		err = fmt.Errorf("parsing BDict starting at offset %d: %w", startingOffset, err)
		return
	}
	if got != 'd' {
		err = fmt.Errorf("parsing BDict starting at offset %d: %w: a bencoded dictionary must start with byte `d`", startingOffset, ErrMalformedInput)
		return
	}

	for {
		yes, err := areWeFinished(d.reader)
		if err = treatEOFAsUnexpected(err); err != nil {
			return nil, fmt.Errorf("parsing BDict starting at offset %d (offset after error triggered: %d): %w", startingOffset, d.Offset(), err)
		}
		if yes {
			break
		}

		key, err := d.DecodeString()
		if err != nil {
			return nil, err
		}

		value, err := d.DecodeAny()
		if err != nil {
			return nil, err
		}

		dict[string(key)] = value
	}

	// no need to check for erros here
	d.reader.ReadByte()

	return
}

// DecodeAny peeks at the next byte in the stream and decodes the appropriate bencoded type:
//   - 'i' -> [BInt] via [Decoder.DecodeInt]
//   - 'l' -> [BList] via [Decoder.DecodeList]
//   - 'd' -> [BDict] via [Decoder.DecodeDict]
//   - '0'-'9' -> [BString] via [Decoder.DecodeString]
//
// Error conditions:
//   - Returns [ErrMalformedInput] if the discriminator byte does not match any valid bencoding type.
//   - Returns any parsing error returned by the dispatched Decode* function.
//   - Returns [io.ErrUnexpectedEOF] if the stream is empty or truncated.
func (d *Decoder) DecodeAny() (any, error) {
	buffer, err := d.reader.Peek(1)
	if err = treatEOFAsUnexpected(err); err != nil {
		return nil, err
	}

	discriminator := buffer[0]

	if discriminator == 'i' {
		return d.DecodeInt()
	}
	if discriminator == 'l' {
		return d.DecodeList()
	}
	if discriminator == 'd' {
		return d.DecodeDict()
	}
	if isDigit(discriminator) {
		return d.DecodeString()
	}

	err = fmt.Errorf("parsing error at offset %d: %w: discriminator %q doesn't correspond to any known type", d.Offset(), ErrMalformedInput, discriminator)
	return nil, err
}

// Checks to see if the next byte is `e` (signaling the
// end of a list or a dictionary). It also forwards
// any io errors received
func areWeFinished(reader *bufio.Reader) (bool, error) {
	buffer, err := reader.Peek(1)
	if err != nil {
		return true, err
	}
	mark := buffer[0]
	return mark == 'e', err
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

func treatEOFAsUnexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
