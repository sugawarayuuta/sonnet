package sonnet

import (
	"errors"
	"reflect"
	"unsafe"
)

type function func(*Decoder, unsafe.Pointer, []byte) error

// compile compiles functions. it supports strings, numbers,
// booleans, objects, arrays.
// errors should be the same as standard library.
func compile(typ *rtype) function {

	var do function
	switch reflect.Kind(typ.kind & kindMask) {
	case reflect.String:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if bytes[0] == 'n' {
				return nil
			} else if bytes[0] != '"' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*string)(ptr) = string(bytes[1 : len(bytes)-1])
			return nil
		}
	case reflect.Int:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*int)(ptr) = int(i64(bytes))
			return nil
		}
	case reflect.Int8:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*int8)(ptr) = int8(i64(bytes))
			return nil
		}
	case reflect.Int16:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*int16)(ptr) = int16(i64(bytes))
			return nil
		}
	case reflect.Int32:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*int32)(ptr) = int32(i64(bytes))
			return nil
		}
	case reflect.Int64:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*int64)(ptr) = i64(bytes)
			return nil
		}
	case reflect.Uint:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*uint)(ptr) = uint(u64(bytes))
			return nil
		}
	case reflect.Uint8:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*uint8)(ptr) = uint8(u64(bytes))
			return nil
		}
	case reflect.Uint16:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*uint16)(ptr) = uint16(u64(bytes))
			return nil
		}
	case reflect.Uint32:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*uint32)(ptr) = uint32(u64(bytes))
			return nil
		}
	case reflect.Uint64:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*uint64)(ptr) = u64(bytes)
			return nil
		}
	case reflect.Float32:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*float32)(ptr) = float32(f64(bytes))
			return nil
		}
	case reflect.Float64:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if (bytes[0] < '0' || bytes[0] > '9') && bytes[0] != '-' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*float64)(ptr) = f64(bytes)
			return nil
		}
	case reflect.Bool:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if bytes[0] != 't' && bytes[0] != 'f' && bytes[0] != 'n' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			*(*bool)(ptr) = bytes[0] == 't'
			return nil
		}
	case reflect.Struct:
		str := typ.fields()
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if bytes[0] == 'n' {
				return nil
			} else if bytes[0] != '{' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}
			for {
				bytes, err := decoder.state(decoder)
				if err != nil {
					return err
				} else if bytes[0] == '}' {
					return nil
				}

				if cache, ok := str.get(bytes[1 : len(bytes)-1]); ok {
					bytes, err = decoder.state(decoder)
					if err != nil {
						return err
					}

					do, ok := load(cache.typ)
					if !ok {
						do = compile(cache.typ)
					}

					err = do(decoder, unsafe.Add(ptr, cache.offset), bytes)
					if err != nil {
						return err
					}
				} else if decoder.disallowUnknownFields {
					return errors.New("unknown field: " + string(bytes))
				} else {
					err = decoder.skip()
					if err != nil {
						return err
					}
				}
			}
		}
	case reflect.Map:
		key, elem := typ.keyAndElemOf()
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if bytes[0] == 'n' {
				return nil
			} else if bytes[0] != '{' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			} else if reflect.Kind(key.kind&kindMask) != reflect.String {
				// only kind that JSON allows for object keys is string.
				// returns an error.
				return errors.New("map key was not string")
			}

			mp := reflect.NewAt(typ.std(), ptr).Elem()

			if mp.IsNil() {
				mp.Set(reflect.MakeMapWithSize(typ.std(), 1))
			}

			for {
				bytes, err := decoder.state(decoder)
				if err != nil {
					return err
				} else if bytes[0] == '}' {
					return nil
				}

				key := string(bytes[1 : len(bytes)-1])
				defer func() {
					_ = key
				}()

				if elem.size > 128 {
					// probably it should fallback to mapassign.
					// it just returns nil for now.
					return nil
				}

				// mapassign_faststr is from the package runtime,
				// not the package reflect. so that i can avoid mem copy.
				val := typ.mapassign_faststr(mp.UnsafePointer(), key)

				bytes, err = decoder.state(decoder)
				if err != nil {
					return err
				}

				do, ok := load(elem)
				if !ok {
					do = compile(elem)
				}

				err = do(decoder, val, bytes)
				if err != nil {
					return err
				}
			}
		}
	case reflect.Slice:
		_, elem := typ.keyAndElemOf()
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			if bytes[0] == 'n' {
				return nil
			} else if bytes[0] != '[' {
				return &UnmarshalTypeError{
					Value:  string(bytes),
					Type:   typ.std(),
					Offset: decoder.InputOffset(),
				}
			}

			header := (*sliceHeader)(ptr)

			for {
				bytes, err := decoder.state(decoder)
				if err != nil {
					return err
				} else if bytes[0] == ']' {
					return nil
				}

				if header.cap < header.len+1 {
					// not enought capacity. grow the slice
					// with the rate of ( previous + 1 ) * 2
					elem.grow(header)
				}

				do, ok := load(elem)
				if !ok {
					// otherwise compile.
					do = compile(elem)
				}

				err = do(decoder, unsafe.Add(header.ptr, elem.size*uintptr(header.len)), bytes)
				if err != nil {
					return err
				}

				header.len++
			}
		}
	case reflect.Interface:
		do = func(decoder *Decoder, ptr unsafe.Pointer, bytes []byte) error {
			in, err := decoder.decodeAnyWith(bytes)
			if err != nil {
				return err
			}
			*(*interface{})(ptr) = in
			return nil
		}
	}
	save(typ, do)
	return do
}
