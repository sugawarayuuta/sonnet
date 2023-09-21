package sonnet

import (
	"encoding/base64"
	"reflect"
	"strconv"
	"sync"
)

func compileSliceDecoder(typ reflect.Type) decoder {
	elm := typ.Elem()
	fnc, ok := decs.get(elm)
	rep := func() {
		if !ok {
			fnc = compileDecoder(elm)
			decs.set(elm, fnc)
		}
	}
	pool := sync.Pool{
		New: func() any {
			val := reflect.New(typ).Elem()
			return &val
		},
	}
	var once sync.Once
	fallback := func(head byte, val reflect.Value, dec *Decoder) error {
		const ull = "ull"
		if head == 'n' {
			part, err := dec.readn(len(ull))
			if err != nil {
				return err
			}
			if string(part) != ull {
				return dec.buildErrSyntax(head, ull, part)
			}
			val.SetZero()
			return nil
		}
		if head != '[' {
			return dec.errUnmarshalType(head, val.Type())
		}
		err := dec.inc()
		if err != nil {
			return err
		}

		once.Do(rep)

		get := pool.Get().(*reflect.Value)
		defer pool.Put(get)
		assign := *get

		for idx := 0; ; idx++ {
			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head == ']' && idx == 0 {
				// already finished in the first iteration,
				// we know this is 0-sized.
				if val.IsNil() {
					val.Set(reflect.MakeSlice(val.Type(), 0, 0))
				}
				val.SetLen(0)
				dec.dep--
				return nil
			}
			if idx >= assign.Cap() {
				assign.Grow(1)              // slices grow one by one, no need to calc.
				assign.SetLen(assign.Cap()) // make sure grown capacity exists as len.
			}
			err = fnc(head, assign.Index(idx), dec)
			if err != nil {
				return err
			}
			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head == ']' {
				if idx >= val.Cap() {
					val.Set(reflect.MakeSlice(val.Type(), idx+1, idx+1))
				}
				val.SetLen(idx + 1) // make sure Copy stops at idx+1.
				reflect.Copy(val, assign)
				dec.dep--
				return nil
			}
			if head != ',' {
				return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after array element")
			}
		}
	}

	if elm.Kind() == reflect.Uint8 {
		return func(head byte, val reflect.Value, dec *Decoder) error {
			const ull = "ull"
			if head == 'n' {
				part, err := dec.readn(len(ull))
				if err != nil {
					return err
				}
				if string(part) != ull {
					return dec.buildErrSyntax(head, ull, part)
				}
				val.SetZero()
				return nil
			}
			if head != '"' {
				return fallback(head, val, dec)
			}
			slice, err := dec.readString()
			if err != nil {
				return err
			}
			std := base64.StdEncoding
			dst := make([]byte, std.DecodedLen(len(slice)))
			length, err := std.Decode(dst, slice)
			if err != nil {
				return err
			}
			val.SetBytes(dst[:length])
			return nil
		}
	}
	return fallback
}

func compileSliceEncoder(typ reflect.Type) encoder {
	elm := typ.Elem()
	ptr := reflect.PointerTo(elm)
	if elm.Kind() == reflect.Uint8 && !ptr.Implements(marshaler) && !ptr.Implements(textMarshaler) {
		return func(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
			if val.IsNil() {
				return append(dst, "null"...), nil
			}
			slice := val.Bytes()
			std := base64.StdEncoding
			src := make([]byte, std.EncodedLen(len(slice)))
			std.Encode(src, slice)
			dst = append(dst, '"')
			dst = append(dst, src...)
			return append(dst, '"'), nil
		}
	}

	fnc, ok := encs.get(elm)
	rep := func() {
		if !ok {
			fnc = compileEncoder(elm, true)
			encs.set(elm, fnc)
		}
	}

	var once sync.Once
	return func(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
		if val.IsNil() {
			return append(dst, "null"...), nil
		}

		once.Do(rep)
		enc.level++
		if enc.level > maxCycles {
			if enc.seen == nil {
				enc.seen = make(map[any]struct{})
			}
			type header struct {
				ptr uintptr
				len int
				cap int
			}
			head := header{
				ptr: val.Pointer(),
				len: val.Len(),
				cap: val.Cap(),
			}
			if _, ok := enc.seen[head]; ok {
				return nil, &UnsupportedValueError{
					Value: val,
					Str:   "encountered a cycle via: " + typ.String(),
				}
			}
			enc.seen[head] = struct{}{}
			defer delete(enc.seen, head)
		}

		dst = append(dst, '[')
		var mid bool
		for idx := 0; idx < val.Len(); idx++ {
			if mid {
				dst = append(dst, ',')
			}
			var err error
			dst, err = fnc(dst, val.Index(idx), enc)
			if err != nil {
				return nil, err
			}
			mid = true
		}
		enc.level--
		return append(dst, ']'), nil
	}
}
