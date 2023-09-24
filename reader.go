package sonnet

import (
	"encoding/binary"
	"errors"
	"github.com/sugawarayuuta/sonnet/internal/arith"
	"github.com/sugawarayuuta/sonnet/internal/mem"
	"io"
	"reflect"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

type (
	// A Decoder reads and decodes JSON values from an input stream.
	Decoder struct {
		buf, sub  []byte
		pos, prev int
		dep       int
		digit     int
		inp       io.Reader
		opt       byte
	}
	accept struct {
		hi, lo byte
	}
)

var (
	lit = binary.LittleEndian
)

var (
	replacer = [1 << 8]byte{
		'"':  '"',
		'\\': '\\',
		'/':  '/',
		'\'': '\'',
		'b':  '\b',
		'f':  '\f',
		'n':  '\n',
		'r':  '\r',
		't':  '\t',
	}
	accepts = [1 << 4]accept{
		{lo: 0x80, hi: (0xbf - 0x80)},
		{lo: 0xa0, hi: (0xbf - 0xa0)},
		{lo: 0x80, hi: (0x9f - 0x80)},
		{lo: 0x90, hi: (0xbf - 0x90)},
		{lo: 0x80, hi: (0x8f - 0x80)},
	}
)

const (
	heads = "" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf1\xf1" +
		"\xf1\xf1\x02\x02\x02\x02\x02\x02" +
		"\x02\x02\x02\x02\x02\x02\x02\x02" +
		"\x02\x02\x02\x02\x02\x02\x02\x02" +
		"\x02\x02\x02\x02\x02\x02\x02\x02" +
		"\x13\x03\x03\x03\x03\x03\x03\x03" +
		"\x03\x03\x03\x03\x03\x23\x03\x03" +
		"\x34\x04\x04\x04\x44\xf1\xf10xF1" +
		"\xf1\xf1\xf1\xf1\xf1\xf1\xf10xF1"
)

const (
	optKeep byte = 1 << iota
	optUnknownFields
	optNumber
)

const (
	maxDep = 10000
)

func (dec *Decoder) inc() error {
	dec.dep++
	if dec.dep > maxDep {
		return errors.New("sonnet: exceeded max depth")
	}
	return nil
}

func (dec *Decoder) errSyntax(msg string) error {
	return &SyntaxError{msg: msg, Offset: int64(dec.prev + dec.pos)}
}

func (dec *Decoder) errUnmarshalType(head byte, typ reflect.Type) error {
	bef := dec.InputOffset()
	err := dec.skip(head)
	if err != nil {
		return err
	}
	var val string
	switch head {
	case '"':
		val = "string"
	case '{':
		val = "object"
	case '[':
		val = "array"
	case 't', 'f':
		val = "bool"
	case 'n':
		val = "null"
	default:
		val = "number"
	}
	return &UnmarshalTypeError{Value: val, Type: typ, Offset: bef}
}

func (dec *Decoder) fill() bool {
	if dec.inp == nil {
		return false
	}
	if cap(dec.buf)-len(dec.buf) >= 1<<10 {
		read, err := dec.inp.Read(dec.buf[len(dec.buf):cap(dec.buf)])
		dec.buf = dec.buf[:len(dec.buf)+read]
		return err == nil && read != 0
	}
	return dec.refill()
}

func (dec *Decoder) refill() bool {
	buf := mem.Get((cap(dec.buf) | 1) << 1)
	pos := dec.pos
	if dec.opt&optKeep != 0 {
		pos = 0
	}
	buf = buf[:copy(buf, dec.buf[pos:])]
	dec.prev += pos
	dec.pos -= pos
	buf, dec.buf = dec.buf, buf
	mem.Put(buf)
	read, err := dec.inp.Read(dec.buf[len(dec.buf):cap(dec.buf)])
	dec.buf = dec.buf[:len(dec.buf)+read]
	return err == nil && read != 0
}

func (dec *Decoder) readn(n int) ([]byte, error) {
	for {
		if dec.pos+n-1 < len(dec.buf) {
			buf := dec.buf[dec.pos : dec.pos+n]
			dec.pos += n
			return buf, nil
		}
		if !dec.fill() {
			return nil, dec.errSyntax("unexpected EOF reading a keyword")
		}
	}
}

func (dec *Decoder) eatSpaces() {
	const spaces uint64 = 1<<' ' | 1<<'\n' | 1<<'\r' | 1<<'\t'
	if dec.pos >= len(dec.buf) || 1<<dec.buf[dec.pos]&spaces != 0 {
		dec.eatSpacesOut()
	}
}

func (dec *Decoder) eatSpacesOut() {
	const spaces uint64 = 1<<' ' | 1<<'\n' | 1<<'\r' | 1<<'\t'
	const x01 uint64 = 0x0101010101010101
	if dec.pos < len(dec.buf) {
		// checked by eatSpaces()
		dec.pos++
	}
	for ; dec.pos+8 <= len(dec.buf); dec.pos += 8 {
		u64 := lit.Uint64(dec.buf[dec.pos:])
		if u64 == x01*' ' || u64 == x01*'\n' || u64 == x01*'\r' || u64 == x01*'\t' {
			continue
		}
		space := arith.NonSpace(u64)
		if space != 8 {
			dec.pos += space
			return
		}
	}
	for ; dec.pos < len(dec.buf) || dec.fill(); dec.pos++ {
		if 1<<dec.buf[dec.pos]&spaces == 0 {
			return
		}
	}
}

func (dec *Decoder) eatString() error {
	for len(dec.buf[dec.pos:]) >= 8 {
		unesc := arith.Escape(lit.Uint64(dec.buf[dec.pos:]))
		dec.pos += unesc
		if unesc != 8 {
			break
		}
	}
	var esc bool
	for {
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return dec.errSyntax("string literal not terminated")
		}
		char := dec.buf[dec.pos]
		dec.pos++
		if esc {
			esc = false
		} else if char == '"' {
			return nil
		} else if char == '\\' {
			esc = true
		}
	}
}

func escape(dst []byte, dec *Decoder) ([]byte, error) {
	const pref = "\\u"
	var esc bool
	var pos int
	for dec.pos+pos < len(dec.buf) || dec.fill() {
		char := dec.buf[dec.pos+pos]
		if esc {
			esc = false
			if rep := replacer[char]; rep != 0 {
				dst = append(dst, rep)
				dec.pos++
				continue
			} else if char != 'u' {
				return nil, dec.errSyntax("invalid escape seqence: \\" + string(char))
			}
			dec.pos++
			if !dec.makeSpace(4) {
				return nil, dec.errSyntax("not enough space to create Unicode code point")
			}
			one, err := dec.hex()
			if err != nil {
				return nil, err
			}
			dec.pos += 4
			if utf16.IsSurrogate(one) {
				if dec.makeSpace(6) && string(dec.buf[dec.pos:dec.pos+len(pref)]) == pref {
					dec.pos += len(pref)
					two, err := dec.hex()
					if err != nil {
						return nil, err
					}
					run := utf16.DecodeRune(one, two)
					if run != utf8.RuneError {
						dec.pos += 4
						dst = utf8.AppendRune(dst, run)
						continue
					}
					dec.pos -= len(pref) // invalid pair, don't consume.
				}
				one = utf8.RuneError
			}
			dst = utf8.AppendRune(dst, one)
		} else if char == '\\' {
			esc = true
			dst = append(dst, dec.buf[dec.pos:dec.pos+pos]...)
			dec.pos += pos + 1
			pos = 0
		} else if char == '"' {
			dst = append(dst, dec.buf[dec.pos:dec.pos+pos]...)
			dec.pos += pos + 1
			pos = 0
			return dst, nil
		} else if char < ' ' {
			return nil, dec.errSyntax("invalid control character: " + strconv.Quote(string(char)))
		} else if char < utf8.RuneSelf {
			pos++
		} else {
			mid := heads[char&^0x80]
			lo := int(mid & 7)
			if !dec.makeSpace(pos+lo) || mid == 0xf1 {
				dst = append(dst, dec.buf[dec.pos:dec.pos+pos]...)
				dec.pos += pos + 1
				pos = 0
				dst = utf8.AppendRune(dst, utf8.RuneError)
				continue
			}
			acc := accepts[mid>>4]
			runes := dec.buf[dec.pos+pos:]
			if runes[1]-acc.lo > acc.hi || lo > 2 && (runes[2]|runes[lo-1])>>6 != 2 {
				dst = append(dst, dec.buf[dec.pos:dec.pos+pos]...)
				dec.pos += pos + 1
				pos = 0
				dst = utf8.AppendRune(dst, utf8.RuneError)
				continue
			}
			pos += lo
		}
	}
	return nil, dec.errSyntax("string literal not terminated")
}

func (dec *Decoder) makeSpace(off int) bool {
	return dec.pos+off <= len(dec.buf) || dec.fill() && dec.pos+off <= len(dec.buf)
}

func (dec *Decoder) readString() ([]byte, error) {
	var pos int
	var err error
	for len(dec.buf[dec.pos+pos:]) >= 8 {
		unesc := arith.Escape(lit.Uint64(dec.buf[dec.pos+pos:]))
		pos += unesc
		if unesc != 8 {
			break
		}
	}
	if dec.pos+pos < len(dec.buf) && dec.buf[dec.pos+pos] == '"' {
		slice := dec.buf[dec.pos : dec.pos+pos]
		dec.pos += pos + 1
		return slice, nil
	}
	dec.sub = append(dec.sub[:0], dec.buf[dec.pos:dec.pos+pos]...)
	dec.pos += pos
	dec.sub, err = escape(dec.sub, dec)
	if err != nil {
		return nil, err
	}
	return dec.sub, nil
}

func (dec *Decoder) hex() (rune, error) {
	var run rune
	for idx := 0; idx < 4; idx++ {
		char := dec.buf[dec.pos+idx]
		switch {
		case '0' <= char && char <= '9':
			char = char - '0'
		case 'a' <= char && char <= 'f':
			char = char - 'a' + 10
		case 'A' <= char && char <= 'F':
			char = char - 'A' + 10
		default:
			return -1, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(char)) + " in \\u hexadecimal character escape")
		}
		run = run<<4 + rune(char)
	}
	return run, nil
}

func (dec *Decoder) readFloat() (float64, error) {
	dec.pos-- // 1 for head.
	neg := dec.buf[dec.pos] == '-'
	var mant uint64
	var pow int64
	if neg {
		dec.pos++
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return 0, dec.errSyntax("JSON number ended with '-'")
		}
		if dec.buf[dec.pos]-'0' >= 10 {
			return 0, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " in numeric literal")
		}
	}
	if dec.buf[dec.pos] == '0' {
		// do nothing as mant is already 0.
		dec.pos++
	} else {
		u64, read := dec.appendUint(0)
		if read == -1 {
			return 0, strconv.ErrRange
		}
		mant = u64
	}
	if dec.pos < len(dec.buf) && dec.buf[dec.pos] == '.' {
		dec.pos++
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return 0, dec.errSyntax("JSON number ended with '.'")
		}
		if dec.buf[dec.pos]-'0' >= 10 {
			return 0, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " after decimal point in numeric literal")
		}
		u64, read := dec.appendUint(mant)
		if read == -1 {
			return 0, strconv.ErrRange
		}
		mant = u64
		pow -= int64(read)
	}
	if dec.pos < len(dec.buf) && dec.buf[dec.pos]|0x20 == 'e' {
		var eneg bool
		dec.pos++
		if (dec.pos < len(dec.buf) || dec.fill()) && (dec.buf[dec.pos] == '+' || dec.buf[dec.pos] == '-') {
			eneg = dec.buf[dec.pos] == '-'
			dec.pos++
		}
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return 0, dec.errSyntax("JSON number ended with 'e' or 'E'")
		}
		if dec.buf[dec.pos]-'0' >= 10 {
			return 0, dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " in exponent of numeric literal")
		}
		enum, read := dec.appendUint(0)
		if read == -1 {
			return 0, strconv.ErrRange
		}
		sign, ok := addSign(enum, eneg)
		if !ok {
			return 0, strconv.ErrRange
		}
		pow += sign
	}
	f64, ok := arith.Lemire64(mant, pow, neg)
	if !ok {
		return 0, strconv.ErrRange
	}
	return f64, nil
}

func (dec *Decoder) eatNumber() error {
	dec.pos-- // 1 for head.
	if dec.buf[dec.pos] == '-' {
		dec.pos++
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return dec.errSyntax("JSON number ended with '-'")
		}
		if dec.buf[dec.pos]-'0' >= 10 {
			return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " in numeric literal")
		}
	}
	if dec.buf[dec.pos] == '0' {
		dec.pos++
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return nil
		}
	} else {
		if dec.eatInteger() {
			return nil
		}
	}
	if dec.buf[dec.pos] == '.' {
		dec.pos++ // the decimal point
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return dec.errSyntax("JSON number ended with '.'")
		}
		if dec.buf[dec.pos]-'0' >= 10 {
			return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " after decimal point in numeric literal")
		}
		dec.pos++ // the required first number character
		if dec.eatInteger() {
			return nil
		}
	}
	// https://blog.cloudflare.com/the-oldest-trick-in-the-ascii-book/
	if dec.buf[dec.pos]|0x20 == 'e' {
		dec.pos++
		if (dec.pos < len(dec.buf) || dec.fill()) && (dec.buf[dec.pos] == '+' || dec.buf[dec.pos] == '-') {
			dec.pos++
		}
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return dec.errSyntax("JSON number ended with 'e' or 'E'")
		}
		if dec.buf[dec.pos]-'0' >= 10 {
			return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(dec.buf[dec.pos])) + " in exponent of numeric literal")
		}
		dec.pos++
		if dec.eatInteger() {
			return nil
		}
	}
	return nil
}

func (dec *Decoder) readInt(head byte) (int64, error) {
	neg := head == '-'
	if !neg {
		dec.pos--
	}
	u64, digit := dec.appendUint(0)
	if digit == -1 {
		return 0, strconv.ErrRange
	}
	sign, ok := addSign(u64, neg)
	if !ok {
		return 0, strconv.ErrRange
	}
	return sign, nil
}

func (dec *Decoder) readUint() (uint64, error) {
	dec.pos--
	u64, digit := dec.appendUint(0)
	if digit == -1 {
		return 0, strconv.ErrRange
	}
	return u64, nil
}

func toInt(src []byte) (int64, error) {
	if len(src) <= 0 {
		return 0, strconv.ErrSyntax
	}
	neg := src[0] == '-'
	if neg {
		src = src[1:]
	}
	u64, err := makeUint(src)
	if err != nil {
		return 0, err
	}
	sign, ok := addSign(u64, neg)
	if !ok {
		return 0, strconv.ErrRange
	}
	return sign, nil
}

func toUint(src []byte) (uint64, error) {
	if len(src) <= 0 {
		return 0, strconv.ErrSyntax
	}
	return makeUint(src)
}

func addSign(u64 uint64, neg bool) (int64, bool) {
	const over = 1 << 63
	if !neg && u64 >= over {
		return 0, false
	}
	if neg && u64 > over {
		return 0, false
	}
	if neg {
		return -int64(u64), true
	}
	return int64(u64), true
}

func makeUint(src []byte) (uint64, error) {
	var dst uint64
	var over bool
	for len(src) >= 8 {
		chunk := lit.Uint64(src)
		if !arith.CanUint64(chunk) {
			return 0, strconv.ErrSyntax
		}
		add := dst*100000000 + arith.Uint64(chunk)
		over, dst = over || add < dst, add
		src = src[8:]
	}
	for len(src) >= 1 {
		num := uint64(src[0] - '0')
		if num > 9 {
			return 0, strconv.ErrSyntax
		}
		add := dst*10 + num
		over, dst = over || add < dst, add
		src = src[1:]
	}
	if over {
		return 0, strconv.ErrRange
	}
	return dst, nil
}

func (dec *Decoder) eatInteger() bool {
	var digit int
	for dec.digit&^7 != 0 && len(dec.buf[dec.pos:]) >= 8 {
		chunk := lit.Uint64(dec.buf[dec.pos:])
		if !arith.CanUint64(chunk) {
			break
		}
		dec.pos += 8
		digit += 8
	}
	for dec.pos < len(dec.buf) || dec.fill() {
		num := uint64(dec.buf[dec.pos] - '0')
		if num > 9 {
			return false
		}
		dec.pos++
		digit++
	}
	dec.digit |= digit
	return true
}

func (dec *Decoder) appendUint(u64 uint64) (uint64, int) {
	var over bool
	var digit int
	for dec.digit&^7 != 0 && len(dec.buf[dec.pos:]) >= 8 {
		chunk := lit.Uint64(dec.buf[dec.pos:])
		if !arith.CanUint64(chunk) {
			break
		}
		add := u64*100000000 + arith.Uint64(chunk)
		over, u64 = over || add < u64, add
		dec.pos += 8
		digit += 8
	}
	for dec.pos < len(dec.buf) || dec.fill() {
		num := uint64(dec.buf[dec.pos] - '0')
		if num > 9 {
			break
		}
		add := u64*10 + num
		over, u64 = over || add < u64, add
		dec.pos++
		digit++
	}
	dec.digit |= digit
	if over {
		return 0, -1
	}
	return u64, digit
}
