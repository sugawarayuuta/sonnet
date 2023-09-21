package sonnet

import (
	"github.com/sugawarayuuta/sonnet/internal/arith"
	"math"
	"reflect"
	"strconv"
	"unicode/utf8"
)

var (
	double = [100]uint16{
		0x3030, 0x3130, 0x3230, 0x3330, 0x3430, 0x3530, 0x3630, 0x3730, 0x3830, 0x3930,
		0x3031, 0x3131, 0x3231, 0x3331, 0x3431, 0x3531, 0x3631, 0x3731, 0x3831, 0x3931,
		0x3032, 0x3132, 0x3232, 0x3332, 0x3432, 0x3532, 0x3632, 0x3732, 0x3832, 0x3932,
		0x3033, 0x3133, 0x3233, 0x3333, 0x3433, 0x3533, 0x3633, 0x3733, 0x3833, 0x3933,
		0x3034, 0x3134, 0x3234, 0x3334, 0x3434, 0x3534, 0x3634, 0x3734, 0x3834, 0x3934,
		0x3035, 0x3135, 0x3235, 0x3335, 0x3435, 0x3535, 0x3635, 0x3735, 0x3835, 0x3935,
		0x3036, 0x3136, 0x3236, 0x3336, 0x3436, 0x3536, 0x3636, 0x3736, 0x3836, 0x3936,
		0x3037, 0x3137, 0x3237, 0x3337, 0x3437, 0x3537, 0x3637, 0x3737, 0x3837, 0x3937,
		0x3038, 0x3138, 0x3238, 0x3338, 0x3438, 0x3538, 0x3638, 0x3738, 0x3838, 0x3938,
		0x3039, 0x3139, 0x3239, 0x3339, 0x3439, 0x3539, 0x3639, 0x3739, 0x3839, 0x3939,
	}
)

const (
	shiftJSON uint64 = 1<<' ' - 1 | 1<<'"'
	shiftHTML uint64 = shiftJSON | 1<<'<' | 1<<'>' | 1<<'&'
)

func appendString(dst []byte, src string, html bool) []byte {
	dst = append(dst, '"')
	mult := len(src) &^ 7
	var total int
	for idx := 0; idx < mult; idx += 8 {
		var pos int
		if html {
			pos = arith.EscapeHTML(uint64String(src[idx:]))
		} else {
			pos = arith.Escape(uint64String(src[idx:]))
		}
		total += pos
		if pos != 8 {
			return appendStringOut(dst, src, html, total)
		}
	}
	for idx := mult; idx < len(src); idx++ {
		char := src[idx]
		if !html && 1<<char&shiftJSON != 0 || html && 1<<char&shiftHTML != 0 || char >= utf8.RuneSelf || char == '\\' {
			return appendStringOut(dst, src, html, total)
		}
		total++
	}
	dst = append(dst, src...)
	return append(dst, '"')
}

func appendStringOut(dst []byte, src string, html bool, idx int) []byte {
	const hex = "0123456789abcdef"
	dst = append(dst, src[:idx]...)
	start := idx
	for idx < len(src) {
		char := src[idx]
		if char < utf8.RuneSelf {
			if (html || 1<<char&shiftJSON == 0) && (!html || 1<<char&shiftHTML == 0) && char != '\\' {
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
		mid := heads[char&^0x80]
		lo := int(mid & 7)
		acc := accepts[mid>>4]
		if len(src)-idx < lo || mid == 0xf1 || src[idx+1]-acc.lo > acc.hi || lo > 2 && (src[idx+2]|src[idx+lo-1])>>6 != 2 {
			if start < idx {
				dst = append(dst, src[start:idx]...)
			}
			dst = append(dst, '\\', 'u', 'f', 'f', 'f', 'd')
			idx++
			start = idx
		} else if char == 0xe2 && idx+2 < len(src) && src[idx+1] == 0x80 && src[idx+2]&^1 == 0xa8 {
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
			dst = append(dst, '\\', 'u', '2', '0', '2', hex[src[idx+2]&0xf])
			idx += lo
			start = idx
		} else {
			idx += lo
		}
	}
	if start < len(src) {
		dst = append(dst, src[start:]...)
	}
	return append(dst, '"')
}

// isValidNumber reports whether num is a valid JSON number literal.
func isValidNumber(num string) bool {
	// This function implements the JSON numbers grammar.
	// See https://tools.ietf.org/html/rfc7159#section-6
	// and https://www.json.org/img/number.png
	if num == "" {
		return false
	}
	// Optional -
	if num[0] == '-' {
		num = num[1:]
		if num == "" {
			return false
		}
	}
	// Digits
	switch {
	default:
		return false

	case num[0] == '0':
		num = num[1:]

	case '1' <= num[0] && num[0] <= '9':
		num = num[1:]
		for len(num) > 0 && '0' <= num[0] && num[0] <= '9' {
			num = num[1:]
		}
	}
	// . followed by 1 or more digits.
	if len(num) >= 2 && num[0] == '.' && '0' <= num[1] && num[1] <= '9' {
		num = num[2:]
		for len(num) > 0 && '0' <= num[0] && num[0] <= '9' {
			num = num[1:]
		}
	}
	// e or E followed by an optional - or + and
	// 1 or more digits.
	if len(num) >= 2 && (num[0] == 'e' || num[0] == 'E') {
		num = num[1:]
		if num[0] == '+' || num[0] == '-' {
			num = num[1:]
			if num == "" {
				return false
			}
		}
		for len(num) > 0 && '0' <= num[0] && num[0] <= '9' {
			num = num[1:]
		}
	}
	// Make sure we are at the end.
	return num == ""
}

func uint64String(str string) uint64 {
	_ = str[7] // bounds check hint to compiler; see golang.org/issue/14808
	return uint64(str[0]) | uint64(str[1])<<8 | uint64(str[2])<<16 | uint64(str[3])<<24 |
		uint64(str[4])<<32 | uint64(str[5])<<40 | uint64(str[6])<<48 | uint64(str[7])<<56
}

func fmtInt(u64 uint64, neg bool) []byte {
	var buf [22]byte
	pos := 22
	for ; u64 >= 100; u64 /= 100 {
		pos -= 2
		lit.PutUint16(buf[pos:], double[u64%100])
	}
	pos -= 2
	lit.PutUint16(buf[pos:], double[u64])
	if u64 < 10 {
		pos++
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return buf[pos:]
}

func appendFloat(dst []byte, f64 float64, bit int) ([]byte, error) {
	if math.IsInf(f64, 0) || math.IsNaN(f64) {
		return nil, &UnsupportedValueError{
			Value: reflect.ValueOf(f64),
			Str:   strconv.FormatFloat(f64, 'g', -1, bit),
		}
	}
	abs := math.Abs(f64)
	format := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if bit == 64 && (abs < 1e-6 || abs >= 1e21) || bit == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			format = 'e'
		}
	}
	dst = strconv.AppendFloat(dst, f64, format, -1, bit)
	if format == 'e' {
		// clean up e-09 to e-9
		n := len(dst)
		if n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst, nil
}
