package sonnet

import (
	"errors"
	"github.com/sugawarayuuta/sonnet/internal/arith"
	"strconv"
)

type (
	compactor struct {
		dst, src    []byte
		read, write int
		dep         int
		html        bool
		prefix      string
		indent      string
	}
)

func (comp *compactor) compact(head byte) error {
	if head == '{' {
		return comp.compactObject()
	}
	if head == '[' {
		return comp.compactArray()
	}
	if head == '"' {
		return comp.eatString()
	}
	keyword := keywords[head]
	if len(keyword) > 0 {
		if comp.read+len(keyword) > len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a keyword")
		}
		word := comp.src[comp.read : comp.read+len(keyword)]
		// it shouldn't allocate for this.
		if string(word) != keyword {
			return comp.buildErrSyntax(head, keyword, word)
		}
		comp.read += len(keyword)
		return nil
	}
	if head-'0' < 10 || head == '-' {
		return comp.eatNumber()
	}
	return comp.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of value")
}

func (comp *compactor) compactArray() error {
	err := comp.inc()
	if err != nil {
		return err
	}
	var mid bool
	for {
		comp.eatSpaces()
		if comp.read >= len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a byte")
		}
		head := comp.src[comp.read]
		comp.read++
		if head == ']' && !mid {
			comp.dep--
			return nil
		}
		if comp.prefix != "" || comp.indent != "" {
			comp.insertNewline()
		}

		err = comp.compact(head)
		if err != nil {
			return err
		}

		comp.eatSpaces()
		if comp.read >= len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a byte")
		}
		head = comp.src[comp.read]
		comp.read++
		if head == ']' {
			comp.dep--
			if comp.prefix != "" || comp.indent != "" {
				comp.insertNewline()
			}
			return nil
		}
		if head != ',' {
			return comp.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after array element")
		}
		mid = true
	}
}

func (comp *compactor) compactObject() error {
	err := comp.inc()
	if err != nil {
		return err
	}
	var mid bool
	for {
		comp.eatSpaces()
		if comp.read >= len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a byte")
		}
		head := comp.src[comp.read]
		comp.read++
		if head == '}' && !mid {
			comp.dep--
			return nil
		}
		if comp.prefix != "" || comp.indent != "" {
			comp.insertNewline()
		}
		if head != '"' {
			return comp.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " looking for beginning of object key string")
		}

		err = comp.eatString()
		if err != nil {
			return err
		}

		comp.eatSpaces()
		if comp.read >= len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a byte")
		}
		head = comp.src[comp.read]
		comp.read++
		if head == '}' && !mid {
			comp.dep--
			return nil
		}
		if head != ':' {
			return comp.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after object key")
		}
		if comp.prefix != "" || comp.indent != "" {
			comp.insertSpace()
		}

		comp.eatSpaces()
		if comp.read >= len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a byte")
		}
		head = comp.src[comp.read]
		comp.read++

		err = comp.compact(head)
		if err != nil {
			return err
		}

		comp.eatSpaces()
		if comp.read >= len(comp.src) {
			return comp.errSyntax("unexpected EOF reading a byte")
		}
		head = comp.src[comp.read]
		comp.read++
		if head == '}' {
			comp.dep--
			if comp.prefix != "" || comp.indent != "" {
				comp.insertNewline()
			}
			return nil
		}
		if head != ',' {
			return comp.errSyntax("invalid character " + strconv.QuoteRune(rune(head)) + " after object key:value pair")
		}
		mid = true
	}
}

func (comp *compactor) inc() error {
	comp.dep++
	if comp.dep > maxDep {
		return errors.New("sonnet: exceeded max depth")
	}
	return nil
}

func (comp *compactor) eatSpaces() {
	const spaces uint64 = 1<<' ' | 1<<'\n' | 1<<'\r' | 1<<'\t'
	if comp.read < len(comp.src) && 1<<comp.src[comp.read]&spaces != 0 {
		comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
		comp.eatSpacesOut()
		comp.write = comp.read
	}
}

func (comp *compactor) eatSpacesOut() {
	// we don't need an inlined fastpath because it's checked by hasSpace()
	const spaces uint64 = 1<<' ' | 1<<'\n' | 1<<'\r' | 1<<'\t'
	const x01 uint64 = 0x0101010101010101
	for ; len(comp.src[comp.read:]) >= 8; comp.read += 8 {
		u64 := lit.Uint64(comp.src[comp.read:])
		if u64 == x01*' ' || u64 == x01*'\n' || u64 == x01*'\r' || u64 == x01*'\t' {
			continue
		}
		space := arith.NonSpace(u64)
		if space != 8 {
			comp.read += space
			return
		}
	}
	for comp.read < len(comp.src) && 1<<comp.src[comp.read]&spaces != 0 {
		comp.read++
	}
}

func (comp *compactor) eatString() error {
	const hex = "0123456789abcdef"
	const shift = 1<<'<' | 1<<'>' | 1<<'&'
	for len(comp.src[comp.read:]) >= 8 {
		var unesc int
		if comp.html {
			unesc = arith.EscapeHTML(lit.Uint64(comp.src[comp.read:]))
		} else {
			unesc = arith.Escape(lit.Uint64(comp.src[comp.read:]))
		}
		comp.read += unesc
		if unesc != 8 {
			break
		}
	}
	var esc bool
	for {
		if comp.read >= len(comp.src) {
			return comp.errSyntax("string literal not terminated")
		}
		char := comp.src[comp.read]
		if comp.html && 1<<char&shift != 0 {
			comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
			comp.dst = append(comp.dst, '\\', 'u', '0', '0', hex[char>>4], hex[char&0xf])
			comp.read++
			comp.write = comp.read
			continue
		}
		if comp.html && char == 0xe2 && comp.read+2 < len(comp.src) {
			fst := comp.src[comp.read+1]
			sec := comp.src[comp.read+2]
			if fst == 0x80 && sec&^1 == 0xa8 {
				comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
				comp.dst = append(comp.dst, '\\', 'u', '0', '0', '2', hex[sec&0xf])
				comp.read += 3
				comp.write = comp.read
				continue
			}
		}
		comp.read++
		if esc {
			esc = false
		} else if char == '"' {
			return nil
		} else if char == '\\' {
			esc = true
		}
	}
}

func (comp *compactor) eatNumber() error {
	const len09 = '9' - '0'
	comp.read--
	if comp.src[comp.read] == '-' {
		comp.read++
		if len(comp.src[comp.read:]) <= 0 {
			return comp.errSyntax("JSON number ended with '-'")
		}
	}
	switch {
	default:
		return comp.errSyntax("JSON number ended with '-'")
	case comp.src[comp.read] == '0':
		comp.read++
	case comp.src[comp.read]-'0' <= len09:
		for len(comp.src[comp.read:]) > 0 && comp.src[comp.read]-'0' <= len09 {
			comp.read++
		}
	}
	if len(comp.src[comp.read:]) >= 2 && comp.src[comp.read] == '.' && comp.src[comp.read+1]-'0' <= len09 {
		comp.read++
		for len(comp.src[comp.read:]) > 0 && comp.src[comp.read]-'0' <= len09 {
			comp.read++
		}
	}
	if len(comp.src[comp.read:]) >= 2 && comp.src[comp.read]|0x20 == 'e' {
		comp.read++
		if comp.src[comp.read] == '+' || comp.src[comp.read] == '-' {
			comp.read++
			if len(comp.src[comp.read:]) <= 0 {
				return comp.errSyntax("JSON number ended with 'e' or 'E'")
			}
		}
		for len(comp.src[comp.read:]) > 0 && comp.src[comp.read]-'0' <= len09 {
			comp.read++
		}
	}
	return nil
}

func (comp *compactor) buildErrSyntax(head byte, exp string, got []byte) error {
	var dif int
	for idx := range got {
		if got[idx] != exp[idx] {
			dif = idx
			break
		}
	}
	return comp.errSyntax("invalid character " +
		strconv.QuoteRune(rune(got[dif])) +
		" in literal " +
		string(head) +
		exp +
		" (expecting " +
		strconv.QuoteRune(rune(exp[dif])) +
		")",
	)
}

func (comp *compactor) errSyntax(msg string) error {
	return &SyntaxError{msg: msg, Offset: int64(comp.read)}
}

func (comp *compactor) insertNewline() {
	comp.read--
	comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
	comp.dst = append(comp.dst, '\n')
	comp.dst = append(comp.dst, comp.prefix...)
	for idx := 0; idx < comp.dep; idx++ {
		comp.dst = append(comp.dst, comp.indent...)
	}
	comp.write = comp.read
	comp.read++
}

func (comp *compactor) insertSpace() {
	comp.dst = append(comp.dst, comp.src[comp.write:comp.read]...)
	comp.dst = append(comp.dst, ' ')
	comp.write = comp.read
}
