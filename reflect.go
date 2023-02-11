package sonnet

import (
	"reflect"
	"unsafe"
)

// rtype is the common implementation of most values.
// It is embedded in other struct types.
// rtype must be kept in sync with ../runtime/type.go:/^type._type.
// this rtype struct is taken from: https://go.dev/src/reflect/type.go
type rtype struct {
	size       uintptr
	ptrdata    uintptr // number of bytes in the type that can contain pointers
	hash       uint32  // hash of type; avoids computation in hash tables
	tflag      uint8   // extra type information flags
	align      uint8   // alignment of variable with this type
	fieldAlign uint8   // alignment of struct field with this type
	kind       uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	equal     func(unsafe.Pointer, unsafe.Pointer) bool
	gcdata    *byte // garbage collection data
	str       int32 // string form
	ptrToThis int32 // type for pointer to this type, may be zero
}

type flag uintptr

const (
	flagKindWidth        = 5 // there are 27 kinds
	flagKindMask    flag = 1<<flagKindWidth - 1
	flagStickyRO    flag = 1 << 5
	flagEmbedRO     flag = 1 << 6
	flagIndir       flag = 1 << 7
	flagAddr        flag = 1 << 8
	flagMethod      flag = 1 << 9
	flagMethodShift      = 10
	flagRO          flag = flagStickyRO | flagEmbedRO
)

const (
	kindDirectIface = 1 << 5
	kindGCProg      = 1 << 6 // Type.gc points to GC program
	kindMask        = (1 << 5) - 1
)

// rtypeAndPointerOf is almost the same as reflect.ValueOf.
// see https://pkg.go.dev/reflect#ValueOf . but it doesn't 
// have flags in it
func rtypeAndPointerOf(param any) (*rtype, unsafe.Pointer) {
	if param != nil {
		val := *(*struct {
			typ *rtype
			ptr unsafe.Pointer
		})(unsafe.Pointer(&param))
		return val.typ, val.ptr
	}
	return nil, nil
}

// flagOf is a helper function of rtypeAndPointerOf.
// it provides a way to make flag out of *rtype.
func (typ *rtype) flagOf() flag {
	flg := flag(typ.kind & kindMask)
	if typ.kind&kindDirectIface == 0 {
		flg |= flagIndir
	}
	return flg
}

// keyAndElemOf returns type of a key and type of a elem.
// the former is nil when it's not a map,
// both are nil when it's not one of array, channel, pointer,
// slice, map.
func (typ *rtype) keyAndElemOf() (*rtype, *rtype) {
	switch reflect.Kind(typ.kind & kindMask) {
	case reflect.Array, reflect.Chan, reflect.Pointer, reflect.Slice:
		return nil, (*struct {
			rtype
			elem *rtype
		})(unsafe.Pointer(typ)).elem
	case reflect.Map:
		typs := (*struct {
			rtype
			key  *rtype
			elem *rtype
		})(unsafe.Pointer(typ))
		return typs.key, typs.elem
	}
	return nil, nil
}
