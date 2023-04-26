package decoder

import (
	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
	"strconv"
)

func (sess *Session) decodeAny(head byte) (any, error) {
	switch head {
	case '{':
		return sess.decodeObjectAny()
	case '[':
		return sess.decodeArrayAny()
	case '"':
		src, err := sess.readString(true)
		if err != nil {
			return nil, err
		}
		return types.String(src), nil
	case 't':
		rue, err := sess.readSize(3)
		if err != nil {
			return nil, err
		} else if types.String(rue) != "rue" {
			return nil, compat.NewSyntaxError(
				"expected true, got: t"+string(rue),
				sess.InputOffset(),
			)
		}
		return true, nil
	case 'f':
		alse, err := sess.readSize(4)
		if err != nil {
			return nil, err
		} else if types.String(alse) != "alse" {
			return nil, compat.NewSyntaxError(
				"expected false, got: f"+string(alse),
				sess.InputOffset(),
			)
		}
		return false, nil
	case 'n':
		ull, err := sess.readSize(3)
		if err != nil {
			return nil, err
		} else if types.String(ull) != "ull" {
			return nil, compat.NewSyntaxError(
				"expected null, got: n"+string(ull),
				sess.InputOffset(),
			)
		}
		return nil, nil
	default:
		if isDigit[head] {
			sess.fix = true
			pos := sess.Pos - 1

			err := sess.consumeNumber()
			if err != nil {
				return nil, err
			}
			sess.fix = false

			src := sess.Buf[pos:sess.Pos]
			if sess.number {
				dst := make([]byte, len(src))
				copy(dst, src)
				str := types.String(dst)
				return compat.Number(str), nil
			}

			return strconv.ParseFloat(types.String(src), 64)
		}
		return nil, compat.NewSyntaxError(
			"unhandled token: "+strconv.Quote(string(head)),
			sess.InputOffset(),
		)
	}
}

func (sess *Session) decodeObjectAny() (map[string]any, error) {

	mapAny := make(map[string]any, 8)

	for {
		head, err := sess.readByte()
		if err != nil {
			return nil, err
		} else if head == '}' && len(mapAny) == 0 {
			return mapAny, nil
		} else if head != '"' {
			return nil, compat.NewSyntaxError(
				"expected a string for an object key",
				sess.InputOffset(),
			)
		}

		src, err := sess.readString(true)
		if err != nil {
			return nil, err
		}

		head, err = sess.readByte()
		if err != nil {
			return nil, err
		} else if head != ':' {
			return nil, compat.NewSyntaxError(
				"expected a colon, got: "+strconv.Quote(string(head)),
				sess.InputOffset(),
			)
		}

		head, err = sess.readByte()
		if err != nil {
			return nil, err
		}

		val, err := sess.decodeAny(head)
		if err != nil {
			return nil, err
		}

		mapAny[types.String(src)] = val

		head, err = sess.readByte()
		if err != nil {
			return nil, err
		} else if head == '}' {
			return mapAny, nil
		} else if head != ',' {
			return nil, compat.NewSyntaxError(
				"expected a comma or a closing }, got: "+strconv.Quote(string(head)),
				sess.InputOffset(),
			)
		}
	}
}

func (sess *Session) decodeArrayAny() ([]any, error) {

	sliceAny := make([]any, 0, 8)

	for {
		head, err := sess.readByte()
		if err != nil {
			return nil, err
		} else if head == ']' && len(sliceAny) == 0 {
			return sliceAny, nil
		}

		val, err := sess.decodeAny(head)
		if err != nil {
			return nil, err
		}

		sliceAny = append(sliceAny, val)

		head, err = sess.readByte()
		if err != nil {
			return nil, err
		} else if head == ']' {
			return sliceAny, nil
		} else if head != ',' {
			return nil, compat.NewSyntaxError(
				"expected a comma or a closing ], got: "+strconv.Quote(string(head)),
				sess.InputOffset(),
			)
		}
	}
}
