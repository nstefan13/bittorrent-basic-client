package bencoding

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

const BStringMaxLength = 64 * 1024 * 1024

type Decoder struct {
	_trackedReader *TrackedReader // only Offset() should use it
	reader         *bufio.Reader
}

func NewDecoder(reader io.Reader) (decoder *Decoder) {
	decoder = &Decoder{}
	decoder._trackedReader = &TrackedReader{reader: reader}
	decoder.reader = bufio.NewReader(decoder._trackedReader)
	return decoder
}

func (d *Decoder) Offset() int {
	return d._trackedReader.offset - d.reader.Buffered()
}

// Decode a bencoded string and returns it as an array of bytes.
// The function expects the string to be at the beginning of the stream
// and respect the BEP3 specifications. Any io.EOF erros encountered during
// parsing are treated as an io.UnexpectedEOF.
// If the length is invalid it returns Atoi-specific
// errors (like strconv.ErrSyntax or strconv.ErrRange) wrapped alongside
// ErrInvalidStringLength.
// To prevent attackers from allocating huge chunks of memory for strings,
// the functions caps the length at 64MB (arbitrary number, not configurable)
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

// Parses an integer in the bencoding format. If the sequence
// doesn't start with an `i` the stream is considered malformed
// and an ErrMalformedInput is returned. If the value of the
// integered is invalid ErrInvalidIntegerValue will be returned,
// optionally alongside some Atoi-specific errors
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

// Parses a list in the bencoding format. If the sequence
// doesn't start with an `l` the stream is considered malformed
// and an ErrMalformedInput is returned. Any errors encountered
// while parsing the elements of the list are returned to the caller
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

// Parses a dictionary in the bencoding format. If the sequence
// doesn't start with an `d` the stream is considered malformed
// and an ErrMalformedInput is returned. Any errors encountered
// while parsing the values and keys of the dictionary are returned
// to the caller
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

// Attempts to decode the beggining of the stream into one of
// the 4 types provided by the bencoding specifications. The
// underlying object it returns is:
//   - of type `BInt` if it decoded an bencoded integer
//   - of type `BString` if it decoded an bencoded string
//   - of type `BList` if it decoded an bencoded list
//   - of type `BDict` if it decoded an bencoded dictionary
//
// This function acts as a router for the other Decode* functions.
// It peeks at the first byte and uses it to guess what data
// might be encoded there. If it cannot deduce the type of data, it
// returns an ErrMalformedInput. If it successfully deduces the type
// of data, it calls the corresponding Decode* function and forwards
// the return values (including any errors)
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
