package decoder

import (
	"encoding"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

func compileMap(typ *types.Type) codec {

	ref := typ.Reflect()
	key, elem := typ.KeyAndElem()

	keyer, prio := compileMapKey(key)
	fun, ok := load(elem)

	stringKind := reflect.Kind(key.Kind&types.KindMask) == reflect.String

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
			*(*unsafe.Pointer)(ptr) = nil
			return nil
		} else if head != '{' {
			return compat.NewUnmarshalTypeError(
				head,
				ref,
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

		mp := reflect.NewAt(ref, ptr).Elem()
		if mp.IsNil() {
			mp.Set(reflect.MakeMapWithSize(ref, 8))
		}

		ptr = mp.UnsafePointer()

		for {
			head, err := sess.readByte()
			if err != nil {
				return err
			} else if head == '}' && mp.Len() == 0 {
				return nil
			} else if head != '"' {
				return compat.NewSyntaxError(
					"expected a string for an object key",
					sess.InputOffset(),
				)
			}

			src, err := sess.readString(true)
			if err != nil {
				return err
			}

			var val unsafe.Pointer
			if stringKind && elem.Size <= 128 && !prio {
				val = typ.MapAssignFaststr(ptr, types.String(src))
			} else {
				keyPtr := key.New()
				err = keyer(src, keyPtr)
				if err != nil {
					return err
				}
				val = typ.MapAssign(ptr, keyPtr)
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

			err = fun(head, val, sess)
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
		}
	}

}

func compileMapKey(typ *types.Type) (func([]byte, unsafe.Pointer) error, bool) {
	kind := reflect.Kind(typ.Kind & types.KindMask)
	ref := typ.Reflect()
	to := reflect.PointerTo(ref)
	if kind != reflect.Pointer && to.Implements(compat.TextUnmarshalerType) {
		return func(src []byte, ptr unsafe.Pointer) error {
			value := reflect.NewAt(ref, ptr)
			unmarshaler := value.Interface()
			return unmarshaler.(encoding.TextUnmarshaler).UnmarshalText(src)
		}, true
	}
	if ref.Implements(compat.TextUnmarshalerType) {
		return func(src []byte, ptr unsafe.Pointer) error {
			value := reflect.NewAt(ref, ptr).Elem()
			if kind == reflect.Pointer && value.IsNil() {
				return nil
			}
			unmarshaler := value.Interface()
			return unmarshaler.(encoding.TextUnmarshaler).UnmarshalText(src)
		}, true
	}
	switch kind {
	case reflect.String:
		return func(src []byte, ptr unsafe.Pointer) error {
			*(*string)(ptr) = types.String(src)
			return nil
		}, false
	case reflect.Int:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btoi(src)
			if err != nil {
				return err
			}
			*(*int)(ptr) = int(i64)
			return nil
		}, false
	case reflect.Int8:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btoi(src)
			if err != nil {
				return err
			}
			*(*int8)(ptr) = int8(i64)
			return nil
		}, false
	case reflect.Int16:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btoi(src)
			if err != nil {
				return err
			}
			*(*int16)(ptr) = int16(i64)
			return nil
		}, false
	case reflect.Int32:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btoi(src)
			if err != nil {
				return err
			}
			*(*int32)(ptr) = int32(i64)
			return nil
		}, false
	case reflect.Int64:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btoi(src)
			if err != nil {
				return err
			}
			*(*int64)(ptr) = int64(i64)
			return nil
		}, false
	case reflect.Uint:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btou(src)
			if err != nil {
				return err
			}
			*(*uint)(ptr) = uint(i64)
			return nil
		}, false
	case reflect.Uint8:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btou(src)
			if err != nil {
				return err
			}
			*(*uint8)(ptr) = uint8(i64)
			return nil
		}, false
	case reflect.Uint16:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btou(src)
			if err != nil {
				return err
			}
			*(*uint16)(ptr) = uint16(i64)
			return nil
		}, false
	case reflect.Uint32:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btou(src)
			if err != nil {
				return err
			}
			*(*uint32)(ptr) = uint32(i64)
			return nil
		}, false
	case reflect.Uint64:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btou(src)
			if err != nil {
				return err
			}
			*(*uint64)(ptr) = uint64(i64)
			return nil
		}, false
	case reflect.Uintptr:
		return func(src []byte, ptr unsafe.Pointer) error {
			i64, err := btou(src)
			if err != nil {
				return err
			}
			*(*uintptr)(ptr) = uintptr(i64)
			return nil
		}, false
	}
	return func(src []byte, ptr unsafe.Pointer) error {
		return typeError(typ.Reflect().String())
	}, true
}
