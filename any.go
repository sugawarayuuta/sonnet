package sonnet

import (
	"strconv"
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
