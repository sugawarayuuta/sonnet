package sonnet

import (
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"
)

func compileStructDecoder(typ reflect.Type) decoder {
	flds := makeFields(typ)

	rep := func() {
		for idx := range flds.flds {
			fld := &flds.flds[idx]
			if fld.dec == nil {
				flw := followType(typ, fld.idxs)
				fld.dec = compileDecoder(flw)
				decs.set(flw, fld.dec)
			}
		}
	}
	var buf []byte
	var atom atomic.Bool
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
		if head != '{' {
			return dec.errUnmarshalType(head, val.Type())
		}
		err := dec.inc()
		if err != nil {
			return err
		}
		once.Do(rep)

		var mid bool
		for {
			dec.eatSpaces()
			if dec.pos >= len(dec.buf) && !dec.fill() {
				return dec.errSyntax("unexpected EOF reading a byte")
			}
			head = dec.buf[dec.pos]
			dec.pos++
			if head == '}' && !mid {
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

			var fld *field
			var ok bool

			if flds.flg&flagDup != 0 {
				hash := hash32(slice, flds.fldsMap.seed)
				tup := flds.fldsMap.tups[hash&flds.fldsMap.mask]
				fld, ok = tup.elm, tup.hash == hash
			}

			if !ok && flds.flg&flagAZ95 != 0 {
				hash := hash32Up(slice, flds.caseMap.seed)
				tup := flds.caseMap.tups[hash&flds.caseMap.mask]
				fld, ok = tup.elm, tup.hash == hash
			} else if !ok && atom.CompareAndSwap(false, true) {
				buf = appendUpper(buf[:0], slice)
				hash := hash32(buf, flds.caseMap.seed)
				tup := flds.caseMap.tups[hash&flds.caseMap.mask]
				fld, ok = tup.elm, tup.hash == hash
				atom.Store(false)
			} else if !ok {
				buf := appendUpper(nil, slice)
				hash := hash32(buf, flds.caseMap.seed)
				tup := flds.caseMap.tups[hash&flds.caseMap.mask]
				fld, ok = tup.elm, tup.hash == hash
			}

			if ok {
				var flw reflect.Value
				if len(fld.idxs) == 1 {
					flw = val.Field(fld.idxs[0])
				} else {
					flw, err = followValue(val, fld.idxs)
					if err != nil {
						return err
					}
				}

				dec.eatSpaces()
				if dec.pos >= len(dec.buf) && !dec.fill() {
					return dec.errSyntax("unexpected EOF reading a byte")
				}
				head = dec.buf[dec.pos]
				dec.pos++
				if head != ':' {
					return dec.errSyntax("invalid character " + strconv.Quote(string(head)) + " after object key")
				}

				dec.eatSpaces()
				if dec.pos >= len(dec.buf) && !dec.fill() {
					return dec.errSyntax("unexpected EOF reading a byte")
				}
				head = dec.buf[dec.pos]
				dec.pos++

				if fld.flg&flagString != 0 && head != 'n' {
					const tmpl = "invalid use of ,string struct tag, trying to unmarshal "
					if head != '"' {
						return fieldError(tmpl + "unquoted value" + " into " + flw.Type().String())
					}

					slice, err := dec.readString()
					if err != nil {
						return err
					}

					temp := Decoder{
						buf:  slice,
						prev: dec.prev + dec.pos - len(slice),
					}

					temp.eatSpaces()
					if len(temp.buf) <= temp.pos {
						return temp.errSyntax("unexpected EOF reading a byte")
					}
					head = temp.buf[temp.pos]
					temp.pos++

					err = fld.dec(head, flw, &temp)
					if err != nil {
						// we could assume this because we already read the entire element.
						err = fieldError(tmpl + strconv.Quote(string(slice)) + " into " + flw.Type().String())
						return err
					}
				} else {
					err = fld.dec(head, flw, dec)
					if err != nil {
						if err, ok := err.(*UnmarshalTypeError); ok {
							if err.Struct == "" {
								err.Struct = typ.Name()
							}
							if err.Field != "" {
								err.Field = "." + err.Field
							}
							err.Field = fld.name + err.Field
						}
						return err
					}
				}
			} else if dec.opt&optUnknownFields != 0 {
				const tmpl = "unknown field "
				return fieldError(tmpl + strconv.Quote(string(slice)))
			} else {
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

				err = dec.skip(head)
				if err != nil {
					return err
				}
			}

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
			mid = true
		}
	}
}

func compileStructEncoder(typ reflect.Type) encoder {
	flds := makeFields(typ)

	rep := func() {
		for idx := range flds.flds {
			fld := &flds.flds[idx]
			if fld.enc == nil {
				flw := followType(typ, fld.idxs)
				fld.enc = compileEncoder(flw, true)
				encs.set(flw, fld.enc)
			}
		}
	}
	var buf []byte
	var atom atomic.Bool
	var once sync.Once
	return func(dst []byte, val reflect.Value, enc *Encoder) ([]byte, error) {
		once.Do(rep)
		dst = append(dst, '{')
		var mid bool
	cont:
		for idx := range flds.flds {
			fld := &flds.flds[idx]
			flw := val
			if len(fld.idxs) == 1 {
				flw = flw.Field(fld.idxs[0])
			} else {
				for _, idx := range fld.idxs {
					if flw.Kind() == reflect.Pointer {
						if flw.IsNil() {
							continue cont
						}
						flw = flw.Elem()
					}
					flw = flw.Field(idx)
				}
			}
			if fld.flg&flagOmitempty != 0 && isEmpty(flw) {
				continue
			}
			key := fld.nameHTML
			if !enc.html {
				key = fld.nameJSON
			}
			if !mid {
				key = key[1:]
			}
			dst = append(dst, key...)
			if fld.flg&flagString != 0 {
				var err error
				if atom.CompareAndSwap(false, true) {
					buf, err = fld.enc(buf[:0], flw, enc)
					if err != nil {
						return nil, err
					}
					if flw.Kind() != reflect.String {
						dst = append(dst, '"')
						dst = append(dst, buf...)
						dst = append(dst, '"')
					} else {
						dst = appendString(dst, string(buf), false)
					}
					atom.Store(false)
				} else {
					buf, err := fld.enc(nil, flw, enc)
					if err != nil {
						return nil, err
					}
					if flw.Kind() != reflect.String {
						dst = append(dst, '"')
						dst = append(dst, buf...)
						dst = append(dst, '"')
					} else {
						dst = appendString(dst, string(buf), false)
					}
				}
			} else {
				var err error
				dst, err = fld.enc(dst, flw, enc)
				if err != nil {
					return nil, err
				}
			}
			mid = true
		}
		return append(dst, '}'), nil
	}
}

func appendUpper(dst []byte, src []byte) []byte {
	const lenAZ = 'Z' - 'A'
	var idx int
	for idx < len(src) {
		char := src[idx]
		if char < utf8.RuneSelf {
			if char&^0x20-'A' <= lenAZ {
				char &^= 0x20
			}
			dst = append(dst, char)
			idx++
			continue
		}
		run, size := utf8.DecodeRune(src[idx:])
		idx += size
		dst = utf8.AppendRune(dst, unicode.ToUpper(run))
	}
	return dst
}

func isEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
