package decoder

import (
	"encoding"
	"math"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/compat"
	"github.com/sugawarayuuta/sonnet/internal/types"
)

type (
	codec func(byte, unsafe.Pointer, *Session) error
)

func compile(typ *types.Type) codec {
	kind := reflect.Kind(typ.Kind & types.KindMask)
	ref := typ.Reflect()
	to := reflect.PointerTo(ref)

	var fun codec
	switch {
	case kind != reflect.Pointer && to.Implements(compat.UnmarshalerType):
		fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
			value := reflect.NewAt(ref, ptr)
			if value.IsNil() {
				return nil
			}

			unmarshaler := value.Interface().(compat.Unmarshaler)

			sess.fix = true
			// -1  to include the head
			off := sess.Pos - 1

			err := sess.skip(head)
			if err != nil {
				return err
			}
			sess.fix = false

			return unmarshaler.UnmarshalJSON(sess.Buf[off:sess.Pos])
		}
	case kind != reflect.Pointer && to.Implements(compat.TextUnmarshalerType):
		fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				return compat.NewUnmarshalTypeError(
					head,
					ref,
					sess.InputOffset(),
				)
			}

			value := reflect.NewAt(ref, ptr)
			if value.IsNil() {
				return nil
			}

			unmarshaler := value.Interface().(encoding.TextUnmarshaler)

			src, err := sess.readString(true)
			if err != nil {
				return err
			}

			return unmarshaler.UnmarshalText(src)
		}
	default:
		switch kind {
		case reflect.String:
			if typ == compat.NumberType {
				fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
					} else if (head < '0' || head > '9') && head != '-' {
						return compat.NewUnmarshalTypeError(
							head,
							ref,
							sess.InputOffset(),
						)
					}

					sess.fix = true
					pos := sess.Pos - 1

					err := sess.consumeNumber()
					if err != nil {
						return err
					}
					sess.fix = false

					src := sess.Buf[pos:sess.Pos]
					dst := make([]byte, len(src))
					copy(dst, src)
					*(*string)(ptr) = types.String(dst)

					return nil
				}
			} else {
				fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
						return compat.NewUnmarshalTypeError(
							head,
							ref,
							sess.InputOffset(),
						)
					}

					src, err := sess.readString(true)
					if err != nil {
						return err
					}

					*(*string)(ptr) = types.String(src)

					return nil
				}
			}
		case reflect.Int:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				i64, err := sess.readInt(head)
				if err != nil {
					return err
				}

				if i64 < math.MinInt || i64 > math.MaxInt {
					return rangeError(strconv.FormatInt(i64, 10))
				}

				*(*int)(ptr) = int(i64)
				return nil
			}
		case reflect.Int8:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				i64, err := sess.readInt(head)
				if err != nil {
					return err
				}

				if i64 < math.MinInt8 || i64 > math.MaxInt8 {
					return rangeError(strconv.FormatInt(i64, 10))
				}

				*(*int8)(ptr) = int8(i64)
				return nil
			}
		case reflect.Int16:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				i64, err := sess.readInt(head)
				if err != nil {
					return err
				}

				if i64 < math.MinInt16 || i64 > math.MaxInt16 {
					return rangeError(strconv.FormatInt(i64, 10))
				}

				*(*int16)(ptr) = int16(i64)
				return nil
			}
		case reflect.Int32:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				i64, err := sess.readInt(head)
				if err != nil {
					return err
				}

				if i64 < math.MinInt32 || i64 > math.MaxInt32 {
					return rangeError(strconv.FormatInt(i64, 10))
				}

				*(*int32)(ptr) = int32(i64)
				return nil
			}
		case reflect.Int64:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				i64, err := sess.readInt(head)
				if err != nil {
					return err
				}

				*(*int64)(ptr) = int64(i64)
				return nil
			}
		case reflect.Uint:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head < '0' || head > '9' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				u64, err := sess.readUint(head)
				if err != nil {
					return err
				}

				if u64 > math.MaxUint {
					return rangeError(strconv.FormatUint(u64, 10))
				}

				*(*uint)(ptr) = uint(u64)
				return nil
			}
		case reflect.Uint8:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head < '0' || head > '9' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				u64, err := sess.readUint(head)
				if err != nil {
					return err
				}

				if u64 > math.MaxUint8 {
					return rangeError(strconv.FormatUint(u64, 10))
				}

				*(*uint8)(ptr) = uint8(u64)
				return nil
			}
		case reflect.Uint16:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head < '0' || head > '9' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				u64, err := sess.readUint(head)
				if err != nil {
					return err
				}

				if u64 > math.MaxUint16 {
					return rangeError(strconv.FormatUint(u64, 10))
				}

				*(*uint16)(ptr) = uint16(u64)
				return nil
			}
		case reflect.Uint32:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head < '0' || head > '9' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				u64, err := sess.readUint(head)
				if err != nil {
					return err
				}

				if u64 > math.MaxUint32 {
					return rangeError(strconv.FormatUint(u64, 10))
				}

				*(*uint32)(ptr) = uint32(u64)
				return nil
			}
		case reflect.Uint64:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head < '0' || head > '9' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				u64, err := sess.readUint(head)
				if err != nil {
					return err
				}

				*(*uint64)(ptr) = uint64(u64)
				return nil
			}
		case reflect.Uintptr:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head < '0' || head > '9' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				u64, err := sess.readUint(head)
				if err != nil {
					return err
				}

				if u64 > math.MaxUint {
					return rangeError(strconv.FormatUint(u64, 10))
				}

				*(*uintptr)(ptr) = uintptr(u64)
				return nil
			}
		case reflect.Float32:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				sess.fix = true
				pos := sess.Pos - 1

				err := sess.consumeNumber()
				if err != nil {
					return err
				}
				sess.fix = false

				src := sess.Buf[pos:sess.Pos]

				f64, err := strconv.ParseFloat(types.String(src), 64)
				if err != nil {
					return err
				}

				if math.Abs(f64) > math.MaxFloat32 {
					return rangeError(strconv.FormatFloat(f64, 'g', -1, 64))
				}

				*(*float32)(ptr) = float32(f64)
				return nil
			}
		case reflect.Float64:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if (head < '0' || head > '9') && head != '-' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				sess.fix = true
				pos := sess.Pos - 1

				err := sess.consumeNumber()
				if err != nil {
					return err
				}
				sess.fix = false

				src := sess.Buf[pos:sess.Pos]

				f64, err := strconv.ParseFloat(types.String(src), 64)
				if err != nil {
					return err
				}

				*(*float64)(ptr) = float64(f64)
				return nil
			}
		case reflect.Bool:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
				} else if head != 't' && head != 'f' {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				if head == 't' {
					rue, err := sess.readSize(3)
					if err != nil {
						return err
					} else if types.String(rue) != "rue" {
						return compat.NewSyntaxError(
							"expected true, got: t"+string(rue),
							sess.InputOffset(),
						)
					}

					*(*bool)(ptr) = true
					return nil
				}

				alse, err := sess.readSize(4)
				if err != nil {
					return err
				} else if types.String(alse) != "alse" {
					return compat.NewSyntaxError(
						"expected false, got: f"+string(alse),
						sess.InputOffset(),
					)
				}

				*(*bool)(ptr) = false
				return nil
			}
		case reflect.Array:
			fun = compileArray(typ)
		case reflect.Slice:
			fun = compileSlice(typ)
		case reflect.Struct:
			fun = compileStruct(typ)
		case reflect.Map:
			fun = compileMap(typ)
		case reflect.Interface:
			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {

				subTyp, subPtr := types.TypeAndPointerOf(*(*any)(ptr))
				// when ptr == subPtr, it's pointing itself.
				// see https://github.com/golang/go/issues/31740.
				if ptr != subPtr &&
					subTyp != nil &&
					reflect.Kind(subTyp.Kind&types.KindMask) == reflect.Pointer {
					_, elem := subTyp.KeyAndElem()

					kind := reflect.Kind(elem.Kind & types.KindMask)

					if subPtr == nil || kind != reflect.Pointer {
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
							temp := (*any)(ptr)
							*temp = nil
							return nil
						}
					}

					fun, ok := load(elem)
					if !ok {
						fun = compile(elem)
						if fun == nil {
							return typeError(elem.Reflect().String())
						}
					}

					return fun(head, subPtr, sess)
				}

				if ref.NumMethod() != 0 {
					return compat.NewUnmarshalTypeError(
						head,
						ref,
						sess.InputOffset(),
					)
				}

				in, err := sess.decodeAny(head)
				if err != nil {
					return err
				}

				*(*any)(ptr) = in
				return nil
			}
		case reflect.Pointer:
			_, elem := typ.KeyAndElem()

			ptrFun, ok := load(elem)

			fun = func(head byte, ptr unsafe.Pointer, sess *Session) error {
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
					temp := (*unsafe.Pointer)(ptr)
					*temp = nil
					return nil
				}

				if !ok {
					ptrFun = compile(elem)
					if ptrFun == nil {
						return typeError(elem.Reflect().String())
					}
					ok = true
				}

				temp := (*unsafe.Pointer)(ptr)
				if *temp == nil {
					*temp = elem.New()
				}

				return ptrFun(head, *temp, sess)
			}
		default:
			return nil
		}
	}
	store(typ, fun)
	return fun
}

func Evaluate(typ *types.Type, ptr unsafe.Pointer, sess *Session) error {
	if typ == nil || ptr == nil || reflect.Kind(typ.Kind&types.KindMask) != reflect.Pointer {
		return compat.NewInvalidUnmarshalError(typ.Reflect())
	}

	_, elem := typ.KeyAndElem()

	fun, ok := load(elem)
	if !ok {
		fun = compile(elem)
		if fun == nil {
			return typeError(elem.Reflect().String())
		}
	}

	head, err := sess.readByte()
	if err != nil {
		return err
	}

	defer subs.Put(&sess.sub)

	return fun(head, ptr, sess)
}
