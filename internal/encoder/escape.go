package encoder

import (
	"math"
	"math/bits"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/types"
)

var (
	escapeTab = [256]bool{
		'"':  true,
		'\\': true,
	}
	escapeHtmlTab = [256]bool{
		'"':  true,
		'\\': true,
		'>':  true,
		'<':  true,
		'&':  true,
	}
)

const (
	hex = "0123456789abcdef"
	lo  = 0x0101010101010101
	hi  = 0x8080808080808080
)

const (
	control   uint64 = lo * ' '
	quote     uint64 = lo * '"'
	backslash uint64 = lo * '\\'
	greater   uint64 = lo * '>'
	less      uint64 = lo * '<'
	ampersand uint64 = lo * '&'
)

func init() {
	// setting these out-of-range characters by looping over them.
	for char := byte(0); char < byte(' '); char++ {
		escapeTab[char] = true
		escapeHtmlTab[char] = true
	}
	char := unicode.MaxASCII + 1
	for ; char <= math.MaxUint8; char++ {
		escapeTab[char] = true
		escapeHtmlTab[char] = true
	}
}

func fallbackIndex(src string, html bool) int {
	header := (*types.StringHeader)(unsafe.Pointer(&src))

	if header.Len >= 8 {
		u64s := *(*[]uint64)(unsafe.Pointer(&types.SliceHeader{
			Ptr: header.Ptr,
			Len: header.Len >> 3,
			Cap: header.Len >> 3,
		}))

		for idx, u64 := range u64s {
			mask := u64 | (u64 - control) | (u64 ^ quote - lo) | (u64 ^ backslash - lo)
			if html {
				mask |= (u64 ^ greater - lo) | (u64 ^ less - lo) | (u64 ^ ampersand - lo)
			}
			if and := mask & hi; and != 0 {
				trail := bits.TrailingZeros64(and)
				return idx<<3 + trail>>3
			}
		}
	}

	if mod := header.Len & 7; mod != 0 {
		length := header.Len - mod
		src = src[length:]
		for idx := range src {
			char := *(*byte)(unsafe.Add(header.Ptr, idx))
			if !html && !escapeTab[char] || html && !escapeHtmlTab[char] {
				continue
			}
			return length + idx
		}
	}

	return -1
}

func appendEscaped(dst []byte, src string, html bool) []byte {

	dst = append(dst, '"')
	idx := fallbackIndex(src, html)
	if idx == -1 {
		dst = append(dst, src...)
		return append(dst, '"')
	}

	return appendEscapedSlow(dst, src, html, idx)
}

func appendEscapedSlow(dst []byte, src string, html bool, idx int) []byte {
	dst = append(dst, src[:idx]...)
	length := len(src)

	start := idx
	for idx < length {
		char := src[idx]

		if char < utf8.RuneSelf {
			if !html && !escapeTab[char] || html && !escapeHtmlTab[char] {
				idx++
				continue
			}

			if start < idx {
				dst = append(dst, src[start:idx]...)
			}

			dst = append(dst, '\\')
			switch char {
			case '\\', '"':
				dst = append(dst, char)
			case '\n':
				dst = append(dst, 'n')
			case '\r':
				dst = append(dst, 'r')
			case '\t':
				dst = append(dst, 't')
			default:
				// This encodes bytes < 0x20 except for \t, \n and \r.
				// If escapeHTML is set, it also escapes <, >, and &
				// because they can lead to security holes when
				// user-controlled strings are rendered into JSON
				// and served to some browsers.
				dst = append(dst, 'u', '0', '0', hex[char>>4], hex[char&0xf])
			}
			idx++
			start = idx
			continue
		}

		run, size := utf8.DecodeRuneInString(src[idx:])
		switch run {
		case utf8.RuneError:
			if start < idx {
				dst = append(dst, src[start:idx]...)
			}
			dst = append(dst, '\\', 'u', 'f', 'f', 'f', 'd')
			idx += size
			start = idx
		case '\u2029', '\u2028':
			// U+2028 is LINE SEPARATOR.
			// U+2029 is PARAGRAPH SEPARATOR.
			// They are both technically valid characters in JSON strings,
			// but don't work in JSONP, which has to be evaluated as JavaScript,
			// and can lead to security holes there. It is valid JSON to
			// escape them, so we do so unconditionally.
			// See http://timelessrepo.com/json-isnt-a-javascript-subset for discussion.
			if start < idx {
				dst = append(dst, src[start:idx]...)
			}
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[run&0xf])
			idx += size
			start = idx
		default:
			idx += size
		}
	}
	if start < length {
		dst = append(dst, src[start:]...)
	}

	return append(dst, '"')
}
