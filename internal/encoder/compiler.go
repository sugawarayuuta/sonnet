package encoder

import (
	"bytes"
	"encoding"
	"reflect"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	codec func([]byte, unsafe.Pointer, Session) ([]byte, error)
)

func compile(typ *types.Type) codec {

	kind := reflect.Kind(typ.Kind & types.KindMask)
	ref := typ.Reflect()
	to := reflect.PointerTo(ref)

	var fun codec
	switch {
	case kind != reflect.Pointer && to.Implements(compat.MarshalerType):
		fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {

			value := reflect.NewAt(ref, ptr)
			if value.IsNil() {
				return append(dst, "null"...), nil
			}

			marshaler := value.Interface().(compat.Marshaler)

			src, err := marshaler.MarshalJSON()
			if err != nil {
				return nil, compat.NewMarshalerError(ref, err, "MarshalJSON")
			}

			var buf bytes.Buffer
			buf.Grow(len(src) / 2)
			err = compact(&buf, src, sess.html)
			if err != nil {
				return nil, compat.NewMarshalerError(ref, err, "MarshalJSON")
			}

			return append(dst, buf.Bytes()...), nil
		}
	case ref.Implements(compat.MarshalerType):
		fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {

			value := reflect.NewAt(ref, ptr).Elem()
			if kind == reflect.Pointer && value.IsNil() {
				return append(dst, "null"...), nil
			}

			marshaler, ok := value.Interface().(compat.Marshaler)
			if !ok {
				return append(dst, "null"...), nil
			}

			src, err := marshaler.MarshalJSON()
			if err != nil {
				return nil, compat.NewMarshalerError(ref, err, "MarshalJSON")
			}

			var buf bytes.Buffer
			buf.Grow(len(src) / 2)
			err = compact(&buf, src, sess.html)
			if err != nil {
				return nil, compat.NewMarshalerError(ref, err, "MarshalJSON")
			}

			return append(dst, buf.Bytes()...), nil
		}
	case kind != reflect.Pointer && to.Implements(compat.TextMarshalerType):
		fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {

			value := reflect.NewAt(ref, ptr)
			if value.IsNil() {
				return append(dst, "null"...), nil
			}

			marshaler := value.Interface().(encoding.TextMarshaler)

			src, err := marshaler.MarshalText()
			if err != nil {
				return nil, compat.NewMarshalerError(ref, err, "MarshalText")
			}

			return appendEscaped(dst, types.String(src), sess.html), nil
		}
	case ref.Implements(compat.TextMarshalerType):
		fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {

			value := reflect.NewAt(ref, ptr).Elem()
			if kind == reflect.Pointer && value.IsNil() {
				return append(dst, "null"...), nil
			}

			marshaler, ok := value.Interface().(encoding.TextMarshaler)
			if !ok {
				return append(dst, "null"...), nil
			}

			src, err := marshaler.MarshalText()
			if err != nil {
				return nil, compat.NewMarshalerError(ref, err, "MarshalText")
			}

			return appendEscaped(dst, types.String(src), sess.html), nil
		}
	default:
		switch kind {
		case reflect.String:
			fun = compileString(typ)
		case reflect.Int:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, itob(int64(*(*int)(ptr)))...), nil
			}
		case reflect.Int8:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, itob(int64(*(*int8)(ptr)))...), nil
			}
		case reflect.Int16:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, itob(int64(*(*int16)(ptr)))...), nil
			}
		case reflect.Int32:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, itob(int64(*(*int32)(ptr)))...), nil
			}
		case reflect.Int64:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, itob(int64(*(*int64)(ptr)))...), nil
			}
		case reflect.Uint:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, utob(uint64(*(*uint)(ptr)))...), nil
			}
		case reflect.Uint8:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, utob(uint64(*(*uint8)(ptr)))...), nil
			}
		case reflect.Uint16:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, utob(uint64(*(*uint16)(ptr)))...), nil
			}
		case reflect.Uint32:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, utob(uint64(*(*uint32)(ptr)))...), nil
			}
		case reflect.Uint64:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, utob(uint64(*(*uint64)(ptr)))...), nil
			}
		case reflect.Uintptr:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return append(dst, utob(uint64(*(*uintptr)(ptr)))...), nil
			}
		case reflect.Float32:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return ftob(dst, float64(*(*float32)(ptr)), 32)
			}
		case reflect.Float64:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				return ftob(dst, float64(*(*float64)(ptr)), 64)
			}
		case reflect.Bool:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				if *(*bool)(ptr) {
					dst = append(dst, "true"...)
				} else {
					dst = append(dst, "false"...)
				}
				return dst, nil
			}
		case reflect.Array:
			fun = compileArray(typ)
		case reflect.Slice:
			fun = compileSlice(typ)
		case reflect.Map:
			fun = compileMap(typ)
		case reflect.Struct:
			fun = compileStruct(typ)
		case reflect.Interface:
			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {

				if *(*unsafe.Pointer)(ptr) == nil {
					return append(dst, "null"...), nil
				}

				typ, ptr := types.TypeAndPointerOf(*(*any)(ptr))

				return Evaluate(dst, typ, &ptr, sess)
			}
		case reflect.Pointer:

			_, elem := typ.KeyAndElem()
			ptrFun, ok := load(elem)

			fun = func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {
				// compile after it launched. so that it can avoid compiling looping
				// objects without stopping.
				// maybe someone has a better idea ...
				if !ok {
					ptrFun = compile(elem)
					if ptrFun == nil {
						return nil, compat.NewUnsupportedTypeError(elem.Reflect())
					}
					ok = true
				}

				if *(*unsafe.Pointer)(ptr) == nil {
					return append(dst, "null"...), nil
				}

				sess.ptrLevel++
				if sess.ptrLevel > startDetectingCyclesAfter {
					// We're a large number of nested ptrEncoder.encode calls deep;
					// start checking if we've run into a pointer cycle.
					if sess.ptrSeen == nil {
						sess.ptrSeen = make(map[unsafe.Pointer]struct{})
					}
					if _, ok := sess.ptrSeen[ptr]; ok {
						return nil, compat.NewUnsupportedValueError(
							reflect.NewAt(ref, ptr).Elem(),
							"encountered a cycle via: "+ref.String(),
						)
					}
					sess.ptrSeen[ptr] = struct{}{}
					defer delete(sess.ptrSeen, ptr)
				}

				var (
					err error
				)

				dst, err = ptrFun(dst, *(*unsafe.Pointer)(ptr), sess)
				if err != nil {
					return nil, err
				}

				sess.ptrLevel--
				return dst, nil
			}
		default:
			return nil
		}
	}
	store(typ, fun)
	return fun
}

func Evaluate(dst []byte, typ *types.Type, ptr *unsafe.Pointer, sess Session) ([]byte, error) {
	if typ == nil {
		return append(dst, "null"...), nil
	}

	fun, ok := load(typ)
	if !ok {
		fun = compile(typ)
		if fun == nil {
			return nil, compat.NewUnsupportedTypeError(typ.Reflect())
		}
	}

	if typ.Kind&types.KindDirectIface == 0 {
		return fun(dst, *ptr, sess)
	}

	return fun(dst, unsafe.Pointer(ptr), sess)
}

//go:linkname compact encoding/json.compact
//go:noescape
func compact(*bytes.Buffer, []byte, bool) error
