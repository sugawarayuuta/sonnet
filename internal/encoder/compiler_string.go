package encoder

import (
	"errors"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileString(typ *types.Type) codec {
	if typ == compat.NumberType {
		return func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
			number := *(*string)(ptr)
			if number == "" {
				number = "0"
			}
			if !isValidNumber(number) {
				return nil, errors.New(`sonnet: invalid number literal: "` + number + `"`)
			}
			return append(dst, number...), nil
		}
	}
	return func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
		return appendEscaped(dst, *(*string)(ptr), sess.html), nil
	}
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
