package sonnet

import (
	"reflect"
	"strconv"
	"sync"
)

func compileArrayDecoder(typ reflect.Type) decoder {
	length := typ.Len()
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
			if err == nil && string(part) != ull {
				err = dec.buildErrSyntax(head, ull, part)
			}
			return err
		}
		if head != '[' {
			return dec.errUnmarshalType(head, val.Type())
		}
		err := dec.inc()
		if err != nil {
			return err
		}

		once.Do(rep)

		val.SetZero()
		for idx := 0; idx < length; idx++ {
			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head == ']' && idx == 0 {
				dec.dep--
				return nil
			}
			err = fnc(head, val.Index(idx), dec)
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
				dec.dep--
				return nil
			}
			if head != ',' {
				return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after array element")
			}
		}
		dec.dep--
		return dec.skipArray(length != 0)
	}
}

func compileArrayEncoder(typ reflect.Type) encoder {
	length := typ.Len()
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
		once.Do(rep)

		dst = append(dst, '[')
		var mid bool
		for idx := 0; idx < length; idx++ {
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
		return append(dst, ']'), nil
	}
}
