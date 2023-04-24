package decoder

import (
	"errors"
	"io"
	"strconv"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/pool"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	reader struct {
		Buf   []byte
		Pos   int
		sub   []byte
		input io.Reader
		err   error
		total int
		fix   bool
		depth int
	}
)

var (
	isSpace = [256]bool{
		' ':  true,
		'\n': true,
		'\t': true,
		'\r': true,
	}
	isDigit = [256]bool{
		'0': true,
		'1': true,
		'2': true,
		'3': true,
		'4': true,
		'5': true,
		'6': true,
		'7': true,
		'8': true,
		'9': true,
		'-': true,
	}
	depthAdder = [256]int{
		'[': +1,
		'{': +1,
		']': -1,
		'}': -1,
	}
)

const (
	// This limits the max nesting depth to prevent stack overflow.
	// This is permitted by https://tools.ietf.org/html/rfc7159#section-9
	maxNestingDepth = 10000
)

func (reader *reader) readByte() (byte, error) {
	for {
		length := len(reader.Buf)
		for reader.Pos < length {
			head := reader.Buf[reader.Pos]
			reader.Pos++
			if !isSpace[head] {
				// [, {: depth++
				// ]. }: depth--
				reader.depth += depthAdder[head]
				if reader.depth > maxNestingDepth {
					return 0, errors.New("sonnet: exceeded max depth")
				}
				return head, nil
			}
		}
		if reader.read() == 0 {
			return 0, compat.NewSyntaxError(
				"unexpected EOF; tried to read a byte",
				reader.InputOffset(),
			)
		}
	}
}

func (reader *reader) readSize(size int) ([]byte, error) {
	for {
		// don't need to skip spaces since it's a part of
		// keywords.
		if len(reader.Buf)-reader.Pos >= size {
			head := reader.Buf[reader.Pos:][:size]
			reader.Pos += size
			return head, nil
		}
		if reader.read() == 0 {
			return nil, compat.NewSyntaxError(
				"unexpected EOF; tried to read a keyword",
				reader.InputOffset(),
			)
		}
	}
}

func (reader *reader) readString(cop bool) ([]byte, error) {
	src := reader.Buf[reader.Pos:]

	var pos int
	for {
		fallback := fallbackIndex(src[pos:])
		if fallback == -1 {
			pos = len(src)
			if reader.read() != 0 {
				src = reader.Buf[reader.Pos:]
				continue
			}
			return nil, compat.NewSyntaxError(
				"sonnet: string literal not terminated",
				reader.InputOffset(),
			)
		}
		pos += fallback
		break
	}

	quote := pos

	var esc bool
out:
	for {
		length := len(src)
		for ; quote < length; quote++ {
			char := src[quote]
			switch {
			case esc:
				esc = false
			case char == '"':
				break out
			case char == '\\':
				esc = true
			}
		}
		if reader.read() != 0 {
			src = reader.Buf[reader.Pos:]
			continue
		}
		return nil, compat.NewSyntaxError(
			"sonnet: string literal not terminated",
			reader.InputOffset(),
		)
	}

	if quote == pos {
		reader.Pos += quote + 1
		if cop {
			if quote > 1024 {
				dst := make([]byte, quote)
				copy(dst, src)
				return dst, nil
			}
			if cap(reader.sub) < quote {
				ptr, ok := subs.Get().(*[]byte)
				if ok && cap(*ptr) >= quote {
					reader.sub = *ptr
				} else {
					reader.sub = make([]byte, 1024)
				}
			}
			dst := reader.sub[:quote]
			reader.sub = reader.sub[quote:]
			copy(dst, src)
			return dst, nil
		}
		return src[:quote], nil
	}

	dst := make([]byte, pos, quote)
	copy(dst, src)
	for {
		switch char := src[pos]; {
		case char == '\\':
			pos++
			if pos == quote {
				return nil, compat.NewSyntaxError(
					"JSON string ended with \\",
					reader.InputOffset(),
				)
			}
			switch char := src[pos]; char {
			default:
				// we don't know what this escape seqence is
				return nil, compat.NewSyntaxError(
					"invalid escape seqence: \\"+string(char),
					reader.InputOffset(),
				)
			case '"', '\\', '/', '\'':
				dst = append(dst, char)
				pos++
			case 'b':
				dst = append(dst, '\b')
				pos++
			case 'f':
				dst = append(dst, '\f')
				pos++
			case 'n':
				dst = append(dst, '\n')
				pos++
			case 'r':
				dst = append(dst, '\r')
				pos++
			case 't':
				dst = append(dst, '\t')
				pos++
			case 'u':
				if pos+5 > quote {
					return nil, compat.NewSyntaxError(
						"sonnet: not enough space to create a rune",
						reader.InputOffset(),
					)
				}
				one := utor(src[pos+1 : pos+5])
				pos += 5
				if utf16.IsSurrogate(one) {
					if pos+6 <= quote && types.String(src[pos:pos+2]) == "\\u" {
						two := utor(src[pos+2 : pos+6])
						run := utf16.DecodeRune(one, two)
						if run != unicode.ReplacementChar {
							// A valid pair; consume.
							pos += 6
							dst = utf8.AppendRune(dst, run)
							break
						}
					}
					// Invalid surrogate; fall back to replacement rune.
					one = unicode.ReplacementChar
				}
				dst = utf8.AppendRune(dst, one)
			}
		case char == '"':
			reader.Pos += pos + 1
			return dst, nil
		case char < ' ':
			// Control
			return nil, compat.NewSyntaxError(
				"sonnet: invalid control character: "+strconv.Quote(string(char)),
				reader.InputOffset(),
			)
		case char < utf8.RuneSelf:
			//ASCII
			dst = append(dst, char)
			pos++
		default:
			// Coerce to well-formed UTF-8.
			run, size := utf8.DecodeRune(src[pos:])
			pos += size
			dst = utf8.AppendRune(dst, run)
		}
	}
}

func (reader *reader) consumeString() error {

	for {
		src := reader.Buf[reader.Pos:]
		fallback := fallbackIndex(src)
		if fallback != -1 {
			reader.Pos += fallback
			break
		}
		reader.Pos += len(src)
		if reader.read() == 0 {
			return compat.NewSyntaxError(
				"sonnet: string literal not terminated",
				reader.InputOffset(),
			)
		}
	}

	var esc bool
	for {
		length := len(reader.Buf)
		for ; reader.Pos < length; reader.Pos++ {
			char := reader.Buf[reader.Pos]
			switch {
			case esc:
				esc = false
			case char == '"':
				reader.Pos++
				return nil
			case char == '\\':
				esc = true
			}
		}
		if reader.read() == 0 {
			return compat.NewSyntaxError(
				"sonnet: string literal not terminated",
				reader.InputOffset(),
			)
		}
	}
}

func (reader *reader) readInt(head byte) (int64, error) {
	var minus bool
	var i64 int64
	if head == '-' {
		minus = true
	} else {
		i64 = int64(head - '0')
	}
	for {
		length := len(reader.Buf)
		for ; reader.Pos < length; reader.Pos++ {
			char := reader.Buf[reader.Pos]
			if char >= '0' && char <= '9' {
				nw := i64*10 + int64(char-'0')
				if nw < i64 {
					return 0, rangeError(strconv.FormatInt(i64, 10))
				}
				i64 = nw
				continue
			}
			if minus {
				return -i64, nil
			}
			return i64, nil
		}
		if reader.read() == 0 {
			if minus {
				return -i64, nil
			}
			return i64, nil
		}
	}
}

func (reader *reader) readUint(head byte) (uint64, error) {
	u64 := uint64(head - '0')
	for {
		length := len(reader.Buf)
		for ; reader.Pos < length; reader.Pos++ {
			char := reader.Buf[reader.Pos]
			if char >= '0' && char <= '9' {
				nw := u64*10 + uint64(char-'0')
				if nw < u64 {
					return 0, rangeError(strconv.FormatUint(u64, 10))
				}
				u64 = nw
				continue
			}
			return u64, nil
		}
		if reader.read() == 0 {
			return u64, nil
		}
	}
}

func (reader *reader) consumeNumber() error {
i64:
	for {
		length := len(reader.Buf)
		for ; reader.Pos < length; reader.Pos++ {
			char := reader.Buf[reader.Pos]
			if char >= '0' && char <= '9' {
				continue
			}
			break i64
		}
		if reader.read() == 0 {
			return nil
		}
	}

	if reader.Buf[reader.Pos] == '.' {
		reader.Pos++

		var count int
	f64:
		for {
			length := len(reader.Buf)
			for ; reader.Pos < length; reader.Pos++ {
				char := reader.Buf[reader.Pos]
				if char >= '0' && char <= '9' {
					count++
					continue
				}
				if count == 0 {
					return compat.NewSyntaxError(
						"number literal ended with .",
						reader.InputOffset(),
					)
				}
				break f64
			}
			if reader.read() == 0 {
				if count == 0 {
					return compat.NewSyntaxError(
						"number literal ended with .",
						reader.InputOffset(),
					)
				}
				return nil
			}
		}
	}

	if reader.Buf[reader.Pos] == 'e' || reader.Buf[reader.Pos] == 'E' {
		reader.Pos++

		var sign int
		var vis bool
	exp:
		for {
			length := len(reader.Buf)
			for ; reader.Pos < length; reader.Pos++ {
				char := reader.Buf[reader.Pos]
				if sign == 0 {
					sign = 1
					if char == '-' {
						sign = -1
					}
					continue
				}
				if char >= '0' && char <= '9' {
					vis = true
					continue
				}
				if !vis {
					return compat.NewSyntaxError(
						"number literal ended with e or E",
						reader.InputOffset(),
					)
				}
				break exp
			}
			if reader.read() == 0 {
				if !vis {
					return compat.NewSyntaxError(
						"number literal ended with e or E",
						reader.InputOffset(),
					)
				}
				return nil
			}
		}
	}

	return nil
}

func (reader *reader) read() int {
	if reader.err != nil || reader.input == nil {
		return 0
	}

	var pos int
	if !reader.fix {
		pos = reader.Pos
	}

	capacity := cap(reader.Buf)
	length := len(reader.Buf)
	if capacity-length+pos >= pool.Min {
		length = copy(reader.Buf, reader.Buf[pos:])

		reader.Pos -= pos
	} else if capacity-length < pool.Min {
		buf := pool.Get((capacity + 1) * 2)
		length = copy(buf[:length], reader.Buf[pos:])
		buf = buf[:length]

		buf, reader.Buf = reader.Buf, buf
		pool.Put(buf)

		reader.total += pos
		reader.Pos -= pos
	}

	read, err := reader.input.Read(reader.Buf[length:cap(reader.Buf)])
	reader.err = err
	reader.Buf = reader.Buf[:length+read]

	return read
}

func (reader *reader) InputOffset() int64 {
	return int64(reader.total + reader.Pos)
}
