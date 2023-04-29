package decoder

import (
	"encoding/base64"
	"reflect"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/pool"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileSlice(typ *types.Type) codec {

	_, elem := typ.KeyAndElem()

	fun, ok := load(elem)

	arrays := pool.ArrayPool{
		Type: elem,
	}
	fallback := func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
			*(*types.SliceHeader)(ptr) = types.SliceHeader{}
			return nil
		} else if head != '[' {
			return compat.NewUnmarshalTypeError(
				head,
				typ.Reflect(),
				sess.InputOffset(),
			)
		}

		if !ok {
			fun = compile(elem)
			if fun == nil {
				return typeError(elem.Reflect().String())
			}
			ok = true
		}

		header := (*types.SliceHeader)(ptr)
		*header = arrays.Get(0)
		header.Len = 0

		for {
			head, err := sess.readByte()
			if err != nil {
				return err
			} else if head == ']' && header.Len == 0 {
				return nil
			}

			if header.Cap < header.Len+1 {
				capacity := header.Cap*2 + 1
				array := arrays.Get(capacity)
				array.Len = header.Len
				elem.TypedSliceCopy(array, *header)
				*header, array = array, *header
				arrays.Put(array)
			}

			ptr := unsafe.Add(header.Ptr, elem.Size*uintptr(header.Len))

			err = fun(head, ptr, sess)
			if err != nil {
				return err
			}

			header.Len++

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
	}

	if reflect.Kind(elem.Kind&types.KindMask) == reflect.Uint8 {
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
			} else if head != '"' {
				// fallback to normal slice parsing
				// if it's not base64 encoded.
				return fallback(head, ptr, sess)
			}

			src, err := sess.readString(false)
			if err != nil {
				return err
			}

			enc := base64.StdEncoding
			length := enc.DecodedLen(len(src))
			dst := make([]byte, length)

			dec, err := enc.Decode(dst, src)
			if err != nil {
				return err
			}

			*(*[]byte)(ptr) = dst[:dec]

			return nil
		}
	}

	return fallback
}
