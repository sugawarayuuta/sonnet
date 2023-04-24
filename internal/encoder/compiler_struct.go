package encoder

import (
	"reflect"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileStruct(typ *types.Type) codec {

	str := makeStructType(typ, make(map[*types.Type]*structType))

	return func(dst []byte, ptr unsafe.Pointer, sess Session) ([]byte, error) {

		dst = append(dst, '{')

		var (
			notFirst bool
		)

		for idx := range str.fields {
			cur := &str.fields[idx]

			ptr := unsafe.Add(ptr, cur.offset)

			if cur.omitempty {
				value := reflect.NewAt(cur.typ.Reflect(), ptr).Elem()
				if isEmptyValue(value) {
					continue
				}
			}

			length := len(dst)

			key := cur.json
			if sess.html {
				key = cur.html
			}

			if !notFirst {
				key = key[1:]
			}

			dst = append(dst, key...)

			if !cur.ok {
				cur.fun = compile(cur.typ)
				if cur.fun == nil {
					return nil, compat.NewUnsupportedTypeError(cur.typ.Reflect())
				}
				cur.ok = true
			}

			if cur.stringify {
				src, err := cur.fun(nil, ptr, sess)
				if err != nil {
					if err == back {
						dst = dst[:length]
						continue
					}
					return nil, err
				}
				dst = appendEscaped(dst, types.String(src), false)
			} else {
				temp, err := cur.fun(dst, ptr, sess)
				if err != nil {
					if err == back {
						dst = dst[:length]
						continue
					}
					return nil, err
				}
				dst = temp
			}

			notFirst = true
		}

		dst = append(dst, '}')

		return dst, nil
	}
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	}
	return false
}
