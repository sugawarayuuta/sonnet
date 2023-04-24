package encoder

import (
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileArray(typ *types.Type) codec {

	array := (*struct {
		types.Type
		Elem *types.Type
		_    *types.Type
		Len  uintptr
	})(unsafe.Pointer(typ))

	fun, ok := load(array.Elem)

	return func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
		// compile after it launched. so that it can avoid compiling looping
		// objects without stopping.
		// maybe someone has a better idea ...
		if !ok {
			fun = compile(array.Elem)
			if fun == nil {
				return nil, compat.NewUnsupportedTypeError(array.Elem.Reflect())
			}
			ok = true
		}

		dst = append(dst, '[')

		var (
			notFirst bool
			err      error
		)

		for idx := uintptr(0); idx < uintptr(array.Len); idx++ {

			if notFirst {
				dst = append(dst, ',')
			}

			dst, err = fun(dst, unsafe.Add(ptr, array.Elem.Size*idx), sess)
			if err != nil {
				return nil, err
			}

			notFirst = true
		}

		dst = append(dst, ']')

		return dst, nil
	}
}
