package sonnet

import (
	"reflect"
	"sync"
)

func compilePointerDecoder(typ reflect.Type) decoder {
	elm := typ.Elem()
	fnc, ok := decs.get(elm)
	rep := func() {
		if !ok {
			fnc = compileDecoder(elm)
			decs.set(elm, fnc)
		}
	}
	var once sync.Once
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
		once.Do(rep)

		if val.IsNil() {
			val.Set(reflect.New(elm))
		}
		return addPointer(fnc(head, val.Elem(), dec))
	}
}

func compilePointerEncoder(typ reflect.Type) encoder {
	elm := typ.Elem()
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

		var err error
		dst, err = fnc(dst, val.Elem(), enc)
		if err != nil {
			return nil, err
		}
		enc.level--
		return dst, nil
	}
}

func compilePointerMarshalerEncoder(typ reflect.Type) encoder {
	var fnc encoder
	rep := func() {
		fnc = compileEncoder(typ, false)
	}
	var once sync.Once
	return func(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
		if !val.CanAddr() {
			once.Do(rep)
			return fnc(dst, val, enc)
		}
		return encodeMarshaler(dst, val.Addr(), enc)
	}
}

func compilePointerTextMarshalerEncoder(typ reflect.Type) encoder {
	var fnc encoder
	rep := func() {
		fnc = compileEncoder(typ, false)
	}
	var once sync.Once
	return func(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
		if !val.CanAddr() {
			once.Do(rep)
			return fnc(dst, val, enc)
		}
		return encodeTextMarshaler(dst, val.Addr(), enc)
	}
}
