package sonnet

import (
	"encoding"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type (
	mapDecoder func([]byte, *Decoder) (reflect.Value, error)
	mapEncoder func(reflect.Value) (string, uint64, bool, error)
)

type (
	pair struct {
		str string
		u64 uint64
		neg bool
		key reflect.Value
		elm reflect.Value
	}
)

func compileMapDecoder(typ reflect.Type) decoder {
	key, elm := typ.Key(), typ.Elem()

	keyFnc := compileMapKeyDecoder(key)
	fnc, ok := decs.get(elm)

	elmVal := reflect.New(elm).Elem()
	// elements get copied when assigning to maps.
	// no need to create it each time, just reset it.

	rep := func() {
		if !ok {
			fnc = compileDecoder(elm)
			decs.set(elm, fnc)
		}
	}
	var once sync.Once
	var atom atomic.Bool
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
		if head != '{' || keyFnc == nil {
			// invalid keys and token mismatch result in the same error.
			return dec.errUnmarshalType(head, val.Type())
		}
		err := dec.inc()
		if err != nil {
			return err
		}

		var assign reflect.Value
		if atom.CompareAndSwap(false, true) {
			defer atom.Store(false)
			assign = elmVal
		} else {
			assign = reflect.New(elm).Elem()
		}

		once.Do(rep)

		if val.IsNil() {
			val.Set(reflect.MakeMap(val.Type()))
		}
		for {
			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head == '}' && val.Len() == 0 {
				dec.dep--
				return nil
			}
			if head != '"' {
				return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of object key string")
			}

			slice, err := dec.readString()
			if err != nil {
				return err
			}
			keyVal, err := keyFnc(slice, dec)
			if err != nil {
				return err
			}

			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head != ':' {
				return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after object key")
			}

			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++

			assign.SetZero()
			err = fnc(head, assign, dec)
			if err != nil {
				return err
			}
			val.SetMapIndex(keyVal, assign)

			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head == '}' {
				dec.dep--
				return nil
			}
			if head != ',' {
				return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after object key:value pair")
			}
		}
	}
}

func compileMapKeyDecoder(typ reflect.Type) mapDecoder {
	const lenInt = 5  // int, int8, int16, int32, int64
	const lenUint = 6 // uint, uint8, uint16, uint32, uint64, uintptr
	kind := typ.Kind()
	ptr := reflect.PointerTo(typ)
	if kind != reflect.Pointer && ptr.Implements(textUnmarshaler) {
		return func(src []byte, dec *Decoder) (reflect.Value, error) {
			val := reflect.New(typ)
			unm := val.Interface().(encoding.TextUnmarshaler)
			return val.Elem(), unm.UnmarshalText(src)
		}
	}
	if typ.Implements(textUnmarshaler) {
		return func(src []byte, dec *Decoder) (reflect.Value, error) {
			val := reflect.New(typ).Elem()
			if kind == reflect.Pointer && val.IsNil() {
				return val, nil
			}
			unm := val.Interface().(encoding.TextUnmarshaler)
			return val, unm.UnmarshalText(src)
		}
	}
	if kind == reflect.String {
		return func(src []byte, dec *Decoder) (reflect.Value, error) {
			val := reflect.ValueOf(string(src))
			if val.Type() != typ {
				val = val.Convert(typ)
			}
			return val, nil
		}
	}
	if kind-reflect.Int < lenInt {
		return func(src []byte, dec *Decoder) (reflect.Value, error) {
			i64, err := toInt(src)
			if err != nil || reflect.Zero(typ).OverflowInt(i64) {
				// errors toInt returns are placeholders;
				// replace them with actual errors.
				err = &UnmarshalTypeError{
					Value:  "number " + string(src),
					Type:   typ,
					Offset: dec.InputOffset() - int64(len(src)),
				}
				return reflect.Value{}, err
			}
			val := reflect.ValueOf(i64)
			if val.Type() != typ {
				val = val.Convert(typ)
			}
			return val, nil
		}
	}
	if kind-reflect.Uint < lenUint {
		return func(src []byte, dec *Decoder) (reflect.Value, error) {
			u64, err := toUint(src)
			if err != nil || reflect.Zero(typ).OverflowUint(u64) {
				err = &UnmarshalTypeError{
					Value:  "number " + string(src),
					Type:   typ,
					Offset: dec.InputOffset() - int64(len(src)),
				}
				return reflect.Value{}, err
			}
			val := reflect.ValueOf(u64)
			if val.Type() != typ {
				val = val.Convert(typ)
			}
			return val, nil
		}
	}
	return nil
}

func compileMapEncoder(typ reflect.Type) encoder {
	const lenInt = 5  // int, int8, int16, int32, int64
	const lenUint = 6 // uint, uint8, uint16, uint32, uint64, uintptr
	key := typ.Key()
	elm := typ.Elem()
	noesc := key.Kind()-reflect.Int < lenInt+lenUint
	noesc = noesc && !key.Implements(textMarshaler)

	keyFnc := compileMapKeyEncoder(key)
	fnc, ok := encs.get(elm)
	rep := func() {
		if !ok {
			fnc = compileEncoder(elm, true)
			encs.set(elm, fnc)
		}
	}
	var cpy []*pair
	var atom atomic.Bool
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
			head := val.Pointer()
			if _, ok := enc.seen[head]; ok {
				return nil, &UnsupportedValueError{
					Value: val,
					Str:   "encountered a cycle via: " + typ.String(),
				}
			}
			enc.seen[head] = struct{}{}
			defer delete(enc.seen, head)
		}

		rng := val.MapRange()
		var prs []*pair
		if atom.CompareAndSwap(false, true) {
			defer atom.Store(false)
			for idx := 0; rng.Next(); idx++ {
				if idx >= len(cpy) {
					// need to be addressable to use later.
					keyVal := reflect.New(key).Elem()
					keyVal.Set(rng.Key())
					str, u64, neg, err := keyFnc(keyVal)
					if err != nil {
						return nil, err
					}
					// the same goes for elements.
					elmVal := reflect.New(elm).Elem()
					elmVal.Set(rng.Value())
					cpy = append(cpy, &pair{
						str: str,
						u64: u64,
						neg: neg,
						key: keyVal,
						elm: elmVal,
					})
					continue
				}
				cpy[idx].key.SetIterKey(rng)
				cpy[idx].elm.SetIterValue(rng)
				str, u64, neg, err := keyFnc(cpy[idx].key)
				if err != nil {
					return nil, err
				}
				cpy[idx].str = str
				cpy[idx].u64 = u64
				cpy[idx].neg = neg
			}
			prs = cpy[:val.Len()]
		} else {
			prs = make([]*pair, val.Len())
			for idx := 0; rng.Next(); idx++ {
				keyVal := rng.Key()
				str, u64, neg, err := keyFnc(keyVal)
				if err != nil {
					return nil, err
				}
				prs[idx] = &pair{
					str: str,
					u64: u64,
					neg: neg,
					key: keyVal,
					elm: rng.Value(),
				}
			}
		}
		if noesc {
			slices.SortFunc(prs, func(fst, sec *pair) int {
				if fst.neg && !sec.neg {
					return -1
				}
				if !fst.neg && sec.neg {
					return 1
				}
				if fst.u64 < sec.u64 {
					return -1
				}
				if fst.u64 > sec.u64 {
					return 1
				}
				return 0
			})
		} else {
			slices.SortFunc(prs, func(fst, sec *pair) int {
				return strings.Compare(fst.str, sec.str)
			})
		}

		dst = append(dst, '{')
		var mid bool
		for _, pr := range prs {
			if mid {
				dst = append(dst, ',')
			}
			if noesc {
				dst = append(dst, '"')
				dst = append(dst, fmtInt(pr.u64, pr.neg)...) // never pr.str!
				dst = append(dst, '"')
			} else {
				dst = appendString(dst, pr.str, enc.html)
			}
			dst = append(dst, ':')
			var err error
			dst, err = fnc(dst, pr.elm, enc)
			if err != nil {
				return nil, err
			}
			mid = true
		}

		enc.level--
		return append(dst, '}'), nil
	}
}

func compileMapKeyEncoder(typ reflect.Type) mapEncoder {
	const lenInt = 5  // int, int8, int16, int32, int64
	const lenUint = 6 // uint, uint8, uint16, uint32, uint64, uintptr
	kind := typ.Kind()
	if kind == reflect.String {
		return func(val reflect.Value) (string, uint64, bool, error) {
			return val.String(), 0, false, nil
		}
	}
	if typ.Implements(textMarshaler) {
		return func(val reflect.Value) (string, uint64, bool, error) {
			if kind == reflect.Pointer && val.IsNil() {
				return "", 0, false, nil
			}
			src, err := val.Interface().(encoding.TextMarshaler).MarshalText()
			return string(src), 0, false, err
		}
	}
	if kind-reflect.Int < lenInt {
		return func(val reflect.Value) (string, uint64, bool, error) {
			i64 := val.Int()
			if i64 < 0 {
				return "", uint64(-i64), true, nil
			}
			return "", uint64(i64), false, nil
		}
	}
	if kind-reflect.Uint < lenUint {
		return func(val reflect.Value) (string, uint64, bool, error) {
			u64 := val.Uint()
			return "", u64, false, nil
		}
	}
	panic("unexpected map key type")
}
