package decoder

import (
	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
	"reflect"
	"unsafe"
)

func compileArray(typ *types.Type) codec {

	array := (*struct {
		types.Type
		Elem *types.Type
		_    *types.Type
		Len  uintptr
	})(unsafe.Pointer(typ))

	fun, ok := load(array.Elem)

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
		} else if head != '[' {
			return compat.NewUnmarshalTypeError(
				head,
				typ.Reflect(),
				sess.InputOffset(),
			)
		}

		if !ok {
			fun = compile(array.Elem)
			if fun == nil {
				return typeError(array.Elem.Reflect().String())
			}
			ok = true
		}

		ref := typ.Reflect()
		val := reflect.NewAt(ref, ptr).Elem()
		val.Set(reflect.Zero(ref))

		for idx := uintptr(0); idx < array.Len; idx++ {
			head, err := sess.readByte()
			if err != nil {
				return err
			} else if head == ']' && idx == 0 {
				return nil
			}

			ptr := unsafe.Add(ptr, array.Elem.Size*idx)

			err = fun(head, ptr, sess)
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
					"expected a comma or a closing ], got: "+string(head),
					sess.InputOffset(),
				)
			}
		}

		return sess.skipArray(array.Len != 0)
	}
}
