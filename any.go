package sonnet

import (
	"reflect"
	"slices"
	"strconv"
	"sync"
)

type (
	sorter struct {
		ents []entry
	}
	entry struct {
		key string
		elm any
	}
)

var (
	sorters = sync.Pool{
		New: func() any {
			return new(sorter)
		},
	}
)

func (dec *Decoder) readAny(head byte) (any, error) {
	if head == '{' {
		return dec.readObjectAny()
	}
	if head == '[' {
		return dec.readArrayAny()
	}
	if head == '"' {
		str, err := dec.readString()
		if err != nil {
			return nil, err
		}
		return string(str), nil
	}
	word := keywords[head]
	if len(word) > 0 {
		part, err := dec.readn(len(word))
		if err != nil {
			return nil, err
		}
		if string(part) != word {
			return nil, dec.buildErrSyntax(head, word, part)
		}
		if head != 'n' {
			return head == 't', nil
		}
		return nil, nil
	}
	if head-'0' < 10 || head == '-' {
		if dec.opt&optNumber != 0 {
			dec.opt |= optKeep
			off := dec.pos - 1
			err := dec.eatNumber()
			if err != nil {
				return nil, err
			}
			dec.opt &^= optKeep
			src := dec.buf[off:dec.pos]
			return Number(src), nil
		}
		dec.opt |= optKeep
		off := dec.pos
		f64, err := dec.readFloat()
		if err == strconv.ErrRange {
			// rare, slow path.
			dec.pos = off
			err = dec.eatNumber()
			if err != nil {
				return 0, err
			}
			f64, err = strconv.ParseFloat(string(dec.buf[off-1:dec.pos]), 64)
		}
		if err != nil {
			return nil, err
		}
		dec.opt &^= optKeep
		return f64, nil
	}
	return nil, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of value")
}

func (dec *Decoder) readObjectAny() (map[string]any, error) {
	mp := make(map[string]any)
	err := dec.inc()
	if err != nil {
		return nil, err
	}
	for {
		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head := dec.buf[dec.pos]
		dec.pos++
		if head == '}' && len(mp) == 0 {
			dec.dep--
			return mp, err
		}
		if head != '"' {
			return nil, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of object key string")
		}

		str, err := dec.readString()
		if err != nil {
			return nil, err
		}
		key := string(str)

		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head = dec.buf[dec.pos]
		dec.pos++
		if head != ':' {
			return nil, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after object key")
		}

		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head = dec.buf[dec.pos]
		dec.pos++

		val, err := dec.readAny(head)
		if err != nil {
			return nil, err
		}
		mp[key] = val

		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head = dec.buf[dec.pos]
		dec.pos++
		if head == '}' {
			dec.dep--
			return mp, err
		}
		if head != ',' {
			return nil, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after object key:value pair")
		}
	}
}

func (dec *Decoder) readArrayAny() ([]any, error) {
	slice := make([]any, 0)
	err := dec.inc()
	if err != nil {
		return nil, err
	}
	for {
		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head := dec.buf[dec.pos]
		dec.pos++
		if head == ']' && len(slice) == 0 {
			dec.dep--
			return slice, err
		}

		val, err := dec.readAny(head)
		if err != nil {
			return nil, err
		}

		slice = append(slice, val)

		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a byte")
		}
		head = dec.buf[dec.pos]
		dec.pos++
		if head == ']' {
			dec.dep--
			return slice, err
		}
		if head != ',' {
			return nil, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after array element")
		}
	}
}

func appendAny(dst []byte, val any, enc *Encoder) ([]byte, error) {
	switch val := val.(type) {
	case nil:
		return append(dst, "null"...), nil
	case string:
		return appendString(dst, val, enc.html), nil
	case float64:
		return appendFloat(dst, val, 64)
	case bool:
		if val {
			return append(dst, "true"...), nil
		}
		return append(dst, "false"...), nil
	case []any:
		return appendArrayAny(dst, val, enc)
	case map[string]any:
		return appendObjectAny(dst, val, enc)
	}
	ref := reflect.ValueOf(val)
	typ := ref.Type()
	fnc, ok := encs.get(typ)
	if !ok {
		fnc = compileEncoder(typ, true)
		encs.set(typ, fnc)
	}
	return fnc(dst, ref, enc)
}

func appendObjectAny(dst []byte, val map[string]any, enc *Encoder) ([]byte, error) {
	if val == nil {
		return append(dst, "null"...), nil
	}
	enc.level++
	if enc.level > maxCycles {
		ref := reflect.ValueOf(val)
		if enc.seen == nil {
			enc.seen = make(map[any]struct{})
		}
		head := ref.Pointer()
		if _, ok := enc.seen[head]; ok {
			return nil, &UnsupportedValueError{
				Value: ref,
				Str:   "encountered a cycle via: " + ref.Type().String(),
			}
		}
		enc.seen[head] = struct{}{}
		defer delete(enc.seen, head)
	}

	srt := sorters.Get().(*sorter)
	if cap(srt.ents) < len(val) {
		srt.ents = make([]entry, len(val))
	} else {
		srt.ents = srt.ents[:len(val)]
	}

	var idx int
	for key, elm := range val {
		ent := &srt.ents[idx]
		ent.key = key
		ent.elm = elm
		idx++
	}
	slices.SortFunc(srt.ents, func(fst, sec entry) int {
		if fst.key < sec.key {
			return -1
		}
		if fst.key > sec.key {
			return 1
		}
		return 0
	})

	dst = append(dst, '{')
	var mid bool
	for _, ent := range srt.ents {
		if mid {
			dst = append(dst, ',')
		}
		dst = appendString(dst, ent.key, enc.html)
		dst = append(dst, ':')
		var err error
		dst, err = appendAny(dst, ent.elm, enc)
		if err != nil {
			return nil, err
		}
		mid = true
	}
	sorters.Put(srt)
	enc.level--
	return append(dst, '}'), nil
}

func appendArrayAny(dst []byte, val []any, enc *Encoder) ([]byte, error) {
	if val == nil {
		return append(dst, "null"...), nil
	}
	enc.level++
	if enc.level > maxCycles {
		ref := reflect.ValueOf(val)
		if enc.seen == nil {
			enc.seen = make(map[any]struct{})
		}
		type header struct {
			ptr uintptr
			len int
			cap int
		}
		head := header{
			ptr: ref.Pointer(),
			len: ref.Len(),
			cap: ref.Cap(),
		}
		if _, ok := enc.seen[head]; ok {
			return nil, &UnsupportedValueError{
				Value: ref,
				Str:   "encountered a cycle via: " + ref.Type().String(),
			}
		}
		enc.seen[head] = struct{}{}
		defer delete(enc.seen, head)
	}

	dst = append(dst, '[')
	var mid bool
	for idx := 0; idx < len(val); idx++ {
		if mid {
			dst = append(dst, ',')
		}
		var err error
		dst, err = appendAny(dst, val[idx], enc)
		if err != nil {
			return nil, err
		}
		mid = true
	}
	enc.level--
	return append(dst, ']'), nil
}
