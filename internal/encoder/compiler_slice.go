package encoder

import (
	"encoding/base64"
	"reflect"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileSlice(typ *types.Type) codec {

	_, elem := typ.KeyAndElem()

	if reflect.Kind(elem.Kind&types.KindMask) == reflect.Uint8 {
		to := reflect.PointerTo(elem.Reflect())
		// byte slice, if it doesn't implement marshalers;
		// follows encoding/json's rules
		if !to.Implements(compat.MarshalerType) && !to.Implements(compat.TextMarshalerType) {
			return func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				if *(*unsafe.Pointer)(ptr) == nil {
					return append(dst, "null"...), nil
				}

				slice := *(*[]byte)(ptr)
				enc := base64.StdEncoding
				length := enc.EncodedLen(len(slice))
				src := make([]byte, length)
				enc.Encode(src, slice)

				dst = append(dst, '"')
				dst = append(dst, src...)
				dst = append(dst, '"')

				return dst, nil
			}
		}
	}

	fun, ok := load(elem)

	return func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
		// compile after it launched. so that it can avoid compiling looping
		// objects without stopping.
		// maybe someone has a better idea ...
		if !ok {
			fun = compile(elem)
			if fun == nil {
				return nil, compat.NewUnsupportedTypeError(elem.Reflect())
			}
			ok = true
		}

		if *(*unsafe.Pointer)(ptr) == nil {
			return append(dst, "null"...), nil
		}

		header := (*types.SliceHeader)(ptr)

		sess.ptrLevel++

		if sess.ptrLevel > startDetectingCyclesAfter {
			// We're a large number of nested ptrEncoder.encode calls deep;
			// start checking if we've run into a pointer cycle.
			if sess.ptrSeen == nil {
				sess.ptrSeen = make(map[unsafe.Pointer]struct{})
			}
			if _, ok := sess.ptrSeen[ptr]; ok {
				return nil, compat.NewUnsupportedValueError(
					reflect.NewAt(typ.Reflect(), ptr).Elem(),
					"encountered a cycle via: "+typ.Reflect().String(),
				)
			}
			sess.ptrSeen[ptr] = struct{}{}
			defer delete(sess.ptrSeen, ptr)
		}

		dst = append(dst, '[')

		var (
			notFirst bool
			err      error
		)

		for idx := uintptr(0); idx < uintptr(header.Len); idx++ {

			if notFirst {
				dst = append(dst, ',')
			}

			dst, err = fun(dst, unsafe.Add(header.Ptr, elem.Size*idx), sess)
			if err != nil {
				return nil, err
			}

			notFirst = true
		}

		dst = append(dst, ']')

		sess.ptrLevel--

		return dst, nil
	}
}
