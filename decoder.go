package sonnet

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"unsafe"
)

type any = interface{}

type Decoder struct {
	scanner               scanner
	state                 func(*Decoder) ([]byte, error)
	stack                 []bool
	useNumber             bool
	disallowUnknownFields bool
}

// NewDecoder creates new decoder with default config.
func NewDecoder(this io.Reader) *Decoder {
	decoder := Decoder{
		state: stateValue,
		stack: make([]bool, 0, 4),
		scanner: scanner{
			reader: reader{
				bytes: make([]byte, 0, 512),
				input: this,
			},
		},
	}
	return &decoder
}

// (*Decoder).Decode reads, scans, decodes data using
// the stream in it in a "standard library" way.
// the param for this should be a pointer and shouldn't be nil
func (decoder *Decoder) Decode(this any) error {

	typ, ptr := rtypeAndPointerOf(this)
	flg := typ.flagOf()

	if reflect.Kind(flg&flagKindMask) != reflect.Pointer {
		return &InvalidUnmarshalError{Type: typ.std()}
	}

	// so that we can nil-check and bring the param
	// for the next func at the same time
	if flg&flagIndir != 0 {
		ptr = *(*unsafe.Pointer)(ptr)
	}

	if ptr == nil {
		return &InvalidUnmarshalError{Type: nil}
	}

	// we don't need the *rtype for the pointer anymore,
	// so override the typ variable with its child
	typ = (*struct {
		rtype
		elem *rtype
	})(unsafe.Pointer(typ)).elem

	bytes, err := decoder.state(decoder)
	if err != nil {
		return err
	}

	do, ok := load(typ)
	if !ok {
		do = compile(typ)
	}

	return do(decoder, ptr, bytes)
}

// (*Decoder).decodeInterface is (*Decoder).decode but for any's
func (decoder *Decoder) decodeAny() (any, error) {
	bytes, err := decoder.state(decoder)
	if err != nil {
		return nil, err
	}

	return decoder.decodeAnyWith(bytes)
}

func (decoder *Decoder) decodeAnyWith(bytes []byte) (any, error) {
	if bytes[0] == '"' {
		return string(bytes[1 : len(bytes)-1]), nil
	} else if (bytes[0] >= '0' && bytes[0] <= '9') || bytes[0] == '-' {
		if decoder.useNumber {
			return Number(bytes), nil
		}
		return f64(bytes), nil // maybe make it so that f64 returns (float64, error).
	} else if bytes[0] == 't' || bytes[0] == 'f' {
		return bytes[0] == 't', nil
	} else if bytes[0] == 'n' {
		return nil, nil
	} else if bytes[0] == '{' {
		return decoder.decodeObjectAny()
	} else if bytes[0] == '[' {
		return decoder.decodeArrayAny()
	}
	return nil, errors.New("unhandled token:" + string(bytes))
}

func (decoder *Decoder) decodeObjectAny() (map[string]any, error) {

	mapAny := make(map[string]any, 1)

	for {
		bytes, err := decoder.state(decoder)
		if err != nil {
			return nil, err
		} else if bytes[0] == '}' {
			return mapAny, nil
		}

		// unquote, JSON strings should only be '"'
		// and they have already validated
		key := string(bytes[1 : len(bytes)-1])

		value, err := decoder.decodeAny()
		if err != nil {
			return nil, err
		}

		mapAny[key] = value
	}
}

func (decoder *Decoder) decodeArrayAny() ([]any, error) {

	sliceAny := make([]any, 0, 1)

	for {
		bytes, err := decoder.state(decoder)
		if err != nil {
			return nil, err
		} else if bytes[0] == ']' {
			return sliceAny, nil
		}

		var value any
		if bytes[0] == '"' {
			value = string(bytes[1 : len(bytes)-1])
		} else if (bytes[0] >= '0' && bytes[0] <= '9') || bytes[0] == '-' {
			if decoder.useNumber {
				value = Number(bytes)
			} else {
				value = f64(bytes)
			}
		} else if bytes[0] == 't' || bytes[0] == 'f' {
			value = bytes[0] == 't'
		} else if bytes[0] == 'n' {
			value = nil
		} else if bytes[0] == '{' {
			value, err = decoder.decodeObjectAny()
			if err != nil {
				return nil, err
			}
		} else if bytes[0] == '[' {
			value, err = decoder.decodeArrayAny()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, errors.New("unhandled token:" + string(bytes))
		}
		sliceAny = append(sliceAny, value)
	}
}

// (*Decoder).skip skips current section. it used when
// structs users provided don't have the field for the object key
func (decoder *Decoder) skip() error {

	bytes, err := decoder.state(decoder)
	if err != nil {
		return err
	}

	switch bytes[0] {
	case '{':
		return decoder.skipObject()
	case '[':
		return decoder.skipArray()
	default:
		return nil
	}
}

// a helper func for above. skips nested arrays
func (decoder *Decoder) skipArray() error {
	for {
		bytes, err := decoder.state(decoder)
		if err != nil {
			return err
		}

		switch bytes[0] {
		case ']':
			return nil
		case '{':
			err := decoder.skipObject()
			if err != nil {
				return err
			}
		case '[':
			err := decoder.skipArray()
			if err != nil {
				return err
			}
		}
	}
}

// a helper func for above, skips nested objects
func (decoder *Decoder) skipObject() error {
	for {
		bytes, err := decoder.state(decoder)
		if err != nil {
			return err
		} else if bytes[0] == '}' {
			return nil
		}

		err = decoder.skip()
		if err != nil {
			return err
		}
	}
}

func (decoder *Decoder) Buffered() io.Reader {
	return bytes.NewReader(decoder.scanner.reader.bytes[decoder.scanner.reader.pos:])
}

func (decoder *Decoder) UseNumber() {
	decoder.useNumber = true
}

func (decoder *Decoder) InputOffset() int64 {
	return int64(decoder.scanner.reader.total + decoder.scanner.reader.pos)
}

func (decoder *Decoder) DisallowUnknownFields() {
	decoder.disallowUnknownFields = true
}

func (decoder *Decoder) More() bool {
	bytes := decoder.scanner.scan()
	more := len(bytes) != 0 && bytes[0] != ']' && bytes[0] != '}'
	decoder.scanner.pos = 0
	return more
}
