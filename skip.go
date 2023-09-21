package sonnet

import (
	"strconv"
)

var (
	keywords = [1 << 8]string{
		't': "rue",
		'f': "alse",
		'n': "ull",
	}
)

func (dec *Decoder) buildErrSyntax(head byte, exp string, got []byte) error {
	var dif int
	for idx := range got {
		if got[idx] != exp[idx] {
			dif = idx
			break
		}
	}
	return dec.errSyntax("invalid character " +
		strconv.QuoteRune(rune(got[dif])) +
		" in literal " +
		string(head) +
		exp +
		" (expecting " +
		strconv.QuoteRune(rune(exp[dif])) +
		")",
	)
}

func (dec *Decoder) skip(head byte) error {
	if head == '{' {
		return dec.skipObject()
	}
	if head == '[' {
		return dec.skipArray(false)
	}
	if head == '"' {
		return dec.eatString()
	}
	keyword := keywords[head]
	if len(keyword) > 0 {
		word, err := dec.readn(len(keyword))
		if err != nil {
			return err
		}
		// it shouldn't allocate for this.
		if string(word) != keyword {
			return dec.buildErrSyntax(head, keyword, word)
		}
		return nil
	}
	if head-'0' < 10 || head == '-' {
		return dec.eatNumber()
	}
	return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of value")
}

func (dec *Decoder) skipArray(mid bool) error {
	err := dec.inc()
	if err != nil {
		return err
	}
	for {
		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return dec.errSyntax("unexpected EOF reading a byte")
		}
		head := dec.buf[dec.pos]
		dec.pos++
		if head == ']' && !mid {
			dec.dep--
			return nil
		}

		err = dec.skip(head)
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
		mid = true
	}
}

func (dec *Decoder) skipObject() error {
	err := dec.inc()
	if err != nil {
		return err
	}
	var mid bool
	for {
		dec.eatSpaces()
		if dec.pos >= len(dec.buf) && !dec.fill() {
			return dec.errSyntax("unexpected EOF reading a byte")
		}
		head := dec.buf[dec.pos]
		dec.pos++
		if head == '}' && !mid {
			dec.dep--
			return nil
		}
		if head != '"' {
			return dec.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of object key string")
		}
		err = dec.eatString()
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

		err = dec.skip(head)
		if err != nil {
			return err
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
