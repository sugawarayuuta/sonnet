package encoder

import (
	"encoding"
	"reflect"
	"sort"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	pair struct {
		key string
		val unsafe.Pointer
	}
	pairs []pair
)

var (
	_ sort.Interface = pairs(nil)
)

func (pairs pairs) Len() int {
	return len(pairs)
}

func (pairs pairs) Swap(one int, two int) {
	pairs[one], pairs[two] = pairs[two], pairs[one]
}

func (pairs pairs) Less(one int, two int) bool {
	return pairs[one].key < pairs[two].key
}

func compileMap(typ *types.Type) codec {

	key, elem := typ.KeyAndElem()

	keyFun := compileMapKey(key)
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

		value := reflect.NewAt(typ.Reflect(), ptr).Elem()
		if value.IsNil() {
			return append(dst, "null"...), nil
		}

		var iter types.Iter
		ptr = value.UnsafePointer()
		pairs := make(pairs, value.Len())
		typ.MapIterInit(ptr, &iter)

		sess.ptrLevel++
		if sess.ptrLevel > startDetectingCyclesAfter {
			// We're a large number of nested ptrEncoder.encode calls deep;
			// start checking if we've run into a pointer cycle.
			if sess.ptrSeen == nil {
				sess.ptrSeen = make(map[unsafe.Pointer]struct{})
			}
			if _, ok := sess.ptrSeen[ptr]; ok {
				return nil, compat.NewUnsupportedValueError(
					value,
					"encountered a cycle via: "+typ.Reflect().String(),
				)
			}
			sess.ptrSeen[ptr] = struct{}{}
			defer delete(sess.ptrSeen, ptr)
		}

		for idx := range pairs {
			key, err := keyFun(iter.Key)
			if err != nil {
				return nil, err
			}

			pairs[idx] = pair{
				key: key,
				val: iter.Elem,
			}

			types.MapIterNext(&iter)
		}

		sort.Sort(pairs)

		dst = append(dst, '{')

		var (
			notFirst bool
			err      error
		)

		for idx := range pairs {
			pair := pairs[idx]

			if notFirst {
				dst = append(dst, ',')
			}

			dst = appendEscaped(dst, pair.key, sess.html)

			dst = append(dst, ':')

			dst, err = fun(dst, pair.val, sess)
			if err != nil {
				return nil, err
			}

			notFirst = true
		}

		dst = append(dst, '}')

		sess.ptrLevel--
		return dst, nil
	}
}

func compileMapKey(typ *types.Type) func(unsafe.Pointer) (string, error) {
	kind := reflect.Kind(typ.Kind & types.KindMask)
	switch kind {
	case reflect.String:
		return func(ptr unsafe.Pointer) (string, error) {
			return *(*string)(ptr), nil
		}
	}
	if typ.Reflect().Implements(compat.TextMarshalerType) {
		return func(ptr unsafe.Pointer) (string, error) {
			value := reflect.NewAt(typ.Reflect(), ptr).Elem()
			if kind == reflect.Pointer && value.IsNil() {
				return "", nil
			}
			marshaler := value.Interface()
			src, err := marshaler.(encoding.TextMarshaler).MarshalText()
			return types.String(src), err
		}
	}
	switch kind {
	case reflect.Int:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(itob(int64(*(*int)(ptr)))), nil
		}
	case reflect.Int8:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(itob(int64(*(*int8)(ptr)))), nil
		}
	case reflect.Int16:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(itob(int64(*(*int16)(ptr)))), nil
		}
	case reflect.Int32:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(itob(int64(*(*int32)(ptr)))), nil
		}
	case reflect.Int64:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(itob(int64(*(*int64)(ptr)))), nil
		}
	case reflect.Uint:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(utob(uint64(*(*uint)(ptr)))), nil
		}
	case reflect.Uint8:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(utob(uint64(*(*uint8)(ptr)))), nil
		}
	case reflect.Uint16:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(utob(uint64(*(*uint16)(ptr)))), nil
		}
	case reflect.Uint32:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(utob(uint64(*(*uint32)(ptr)))), nil
		}
	case reflect.Uint64:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(utob(uint64(*(*uint64)(ptr)))), nil
		}
	case reflect.Uintptr:
		return func(ptr unsafe.Pointer) (string, error) {
			return types.String(utob(uint64(*(*uintptr)(ptr)))), nil
		}
	}
	panic("unexpected map key type")
}
