package decoder

import (
	"io"
	"strconv"
	"sync"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	Session struct {
		unknownFields bool
		number        bool
		reader
	}
)

var (
	subs sync.Pool
)

func NewSession(input io.Reader, bytes []byte) Session {
	sess := Session{
		reader: reader{
			Buf:   bytes,
			input: input,
		},
	}
	return sess
}

func (sess *Session) DisallowUnknownFields() {
	sess.unknownFields = true
}

func (sess *Session) UseNumber() {
	sess.number = true
}

func (sess *Session) More() bool {
	head, err := sess.readByte()
	more := err == nil && head != ']' && head != '}'
	sess.Pos--
	return more
}

func (sess *Session) Token() (compat.Token, error) {
	head, err := sess.readByte()
	if err != nil {
		return nil, err
	}

	switch head {
	case '{', '}', '[', ']':
		return compat.Delim(head), nil
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
	case '"':
		src, err := sess.readString(true)
		if err != nil {
			return nil, err
		}
		return types.String(src), nil
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