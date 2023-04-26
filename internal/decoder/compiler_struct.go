package decoder

import (
	"errors"
	"strconv"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileStruct(typ *types.Type) codec {

	str := makeStructType(typ, make(map[*types.Type]*structType))

	return func(head byte, ptr unsafe.Pointer, sess *Session) error {
		if head == 'n' {
			ull, err := sess.readSize(3)
			if err != nil {
				return err
			} else if types.String(ull) != "ull" {
				return compat.NewSyntaxError(
					"expected null, got: n"+string(ull),
					sess.InputOffset(),
				)
			}
			return nil
		} else if head != '{' {
			return compat.NewUnmarshalTypeError(
				head,
				typ.Reflect(),
				sess.InputOffset(),
			)
		}

		var notFirst bool
		for {
			head, err := sess.readByte()
			if err != nil {
				return err
			} else if head == '}' && !notFirst {
				return nil
			} else if head != '"' {
				return compat.NewSyntaxError(
					"expected a string for an object key, got: "+strconv.Quote(string(head)),
					sess.InputOffset(),
				)
			}

			src, err := sess.readString(false)
			if err != nil {
				return err
			}

			var (
				field *structField
				ok    bool
			)

			if str.duplicate {
				hash := fnv(str.fieldsTab.seed, types.String(src))
				pos := hash & str.fieldsTab.mask
				field = &str.fieldsTab.pairs[pos].field
				ok = str.fieldsTab.pairs[pos].hash == hash
			}

			if !ok && str.ascii {
				// unrolled FNV1a
				// manually inlined (automatic inline wouldn't work)
				hash := offset64 ^ str.fieldCaseTab.seed
				for len(src) >= 8 {
					hash = (hash ^ uint64(toLower[src[0]])) * prime64
					hash = (hash ^ uint64(toLower[src[1]])) * prime64
					hash = (hash ^ uint64(toLower[src[2]])) * prime64
					hash = (hash ^ uint64(toLower[src[3]])) * prime64
					hash = (hash ^ uint64(toLower[src[4]])) * prime64
					hash = (hash ^ uint64(toLower[src[5]])) * prime64
					hash = (hash ^ uint64(toLower[src[6]])) * prime64
					hash = (hash ^ uint64(toLower[src[7]])) * prime64
					src = src[8:]
				}
				if len(src) >= 4 {
					hash = (hash ^ uint64(toLower[src[0]])) * prime64
					hash = (hash ^ uint64(toLower[src[1]])) * prime64
					hash = (hash ^ uint64(toLower[src[2]])) * prime64
					hash = (hash ^ uint64(toLower[src[3]])) * prime64
					src = src[4:]
				}
				if len(src) >= 2 {
					hash = (hash ^ uint64(toLower[src[0]])) * prime64
					hash = (hash ^ uint64(toLower[src[1]])) * prime64
					src = src[2:]
				}
				if len(src) >= 1 {
					hash = (hash ^ uint64(toLower[src[0]])) * prime64
				}
				pos := hash & str.fieldCaseTab.mask
				field = &str.fieldCaseTab.pairs[pos].field
				ok = str.fieldCaseTab.pairs[pos].hash == hash
			} else if !ok {
				src = appendToLower(sess.sub[:0], src)
				hash := fnv(str.fieldCaseTab.seed, types.String(src))
				pos := hash & str.fieldCaseTab.mask
				field = &str.fieldCaseTab.pairs[pos].field
				ok = str.fieldCaseTab.pairs[pos].hash == hash
			}

			if ok {
				if !field.ok {
					field.fun = compile(field.typ)
					if field.fun == nil {
						return typeError(field.typ.Reflect().String())
					}
					field.ok = true
				}

				sess := sess
				// HACK: so that stringify doesn't override
				// the original sess variable
				// let me know if you have a better idea...
				// see line 102~ for more.

				head, err := sess.readByte()
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

				if field.stringify && head != 'n' {
					// we should ignore the stringify option
					// if the value is null. this shouldn't be an error.
					if head != '"' {
						return errors.New("sonnet: invalid use of ,string option")
					}

					src, err := sess.readString(false)
					if err != nil {
						return err
					}

					// create a new session, for this instruction only.
					temp := NewSession(nil, src)

					// content of the unstringified string literal.
					head, err = temp.readByte()
					if err != nil {
						return err
					}

					// don't override sess if there are more reads
					// in the same scope.
					// in this case, there aren't.
					sess = &temp
				}

				ptr := unsafe.Add(ptr, field.offset)

				err = field.fun(head, ptr, sess)
				if err != nil {
					return err
				}
			} else if sess.unknownFields {
				return fieldError(src)
			} else {
				head, err := sess.readByte()
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
}
