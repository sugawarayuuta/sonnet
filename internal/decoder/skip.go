package decoder

import (
	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
	"strconv"
)

func (sess *Session) skip(head byte) error {
	switch head {
	case '{':
		return sess.skipObject(false)
	case '[':
		return sess.skipArray(false)
	case '"':
		err := sess.consumeString()
		if err != nil {
			return err
		}
	case 't':
		rue, err := sess.readSize(3)
		if err != nil {
			return err
		} else if types.String(rue) != "rue" {
			return compat.NewSyntaxError(
				"expected true, got: t"+string(rue),
				sess.InputOffset(),
			)
		}
	case 'f':
		alse, err := sess.readSize(4)
		if err != nil {
			return err
		} else if types.String(alse) != "alse" {
			return compat.NewSyntaxError(
				"expected false, got: f"+string(alse),
				sess.InputOffset(),
			)
		}
	case 'n':
		ull, err := sess.readSize(3)
		if err != nil {
			return err
		} else if types.String(ull) != "ull" {
			return compat.NewSyntaxError(
				"expected null, got: n"+string(ull),
				sess.InputOffset(),
			)
		}
	default:
		if isDigit[head] {
			return sess.consumeNumber()
		}
		return compat.NewSyntaxError(
			"unhandled token: "+strconv.Quote(string(head)),
			sess.InputOffset(),
		)
	}
	return nil
}

// a helper func for above. skips nested arrays
func (sess *Session) skipArray(notFirst bool) error {

	for {
		head, err := sess.readByte()
		if err != nil {
			return err
		} else if head == ']' && !notFirst {
			return nil
		}

		err = sess.skip(head)
		if err != nil {
			return err
		}

		head, err = sess.readByte()
		if err != nil {
			return err
		} else if head == ']' {
			return nil
		} else if head != ',' {
			return compat.NewSyntaxError(
				"expected a comma or a closing ], got: "+strconv.Quote(string(head)),
				sess.InputOffset(),
			)
		}

		notFirst = true
	}
}

// a helper func for above, skips nested objects
func (sess *Session) skipObject(notFirst bool) error {

	for {
		head, err := sess.readByte()
		if err != nil {
			return err
		} else if head == '}' && !notFirst {
			return nil
		} else if head != '"' {
			return compat.NewSyntaxError(
				"expected a string for an object key",
				sess.InputOffset(),
			)
		}

		err = sess.consumeString()
		if err != nil {
			return err
		}

		head, err = sess.readByte()
		if err != nil {
			return err
		} else if head != ':' {
			return compat.NewSyntaxError(
				"expected a colon, got: "+strconv.Quote(string(head)),
				sess.InputOffset(),
			)
		}

		head, err = sess.readByte()
		if err != nil {
			return err
		}

		err = sess.skip(head)
		if err != nil {
			return err
		}

		head, err = sess.readByte()
		if err != nil {
			return err
		} else if head == '}' {
			return nil
		} else if head != ',' {
			return compat.NewSyntaxError(
				"expected a comma or a closing }, got: "+strconv.Quote(string(head)),
				sess.InputOffset(),
			)
		}

		notFirst = true
	}
}
