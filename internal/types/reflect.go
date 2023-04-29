package types

import (
	"reflect"
	"unsafe"
)

// Type is the common implementation of most values.
// It is embedded in other struct types.
// Type must be kept in sync with ../runtime/type.go:/^type._type.
// this Type struct is taken from: https://go.dev/src/reflect/type.go
type Type struct {
	Size uintptr
	_    uintptr // number of bytes in the type that can contain pointers
	_    uint32  // hash of type; avoids computation in hash tables
	_    uint8   // extra type information flags
	_    uint8   // alignment of variable with this type
	_    uint8   // alignment of struct field with this type
	Kind uint8   // enumeration for C
	// function for comparing objects of this type
	// (ptr to object A, ptr to object B) -> ==?
	_ func(unsafe.Pointer, unsafe.Pointer) bool
	_ *byte // garbage collection data
	_ int32 // string form
	_ int32 // type for pointer to this type, may be zero
}

// StringHeader represents reflect.StringHeader but with unsafe.Pointer
// instead of uintptr in the first field
type StringHeader struct {
	Ptr unsafe.Pointer
	Len int
}

// SliceHeader represents reflect.SliceHeader but with unsafe.Pointer
// instead of uintptr in the first field
type SliceHeader struct {
	Ptr unsafe.Pointer
	Len int
	Cap int
}

// A hash iteration structure.
// If you modify Iter, also change cmd/compile/internal/reflectdata/reflect.go
// and reflect/value.go to match the layout of this structure.
type Iter struct {
	Key  unsafe.Pointer // Must be in first position.  Write nil to indicate iteration end (see cmd/compile/internal/walk/range.go).
	Elem unsafe.Pointer // Must be in second position (see cmd/compile/internal/walk/range.go).
	_    unsafe.Pointer
	_    unsafe.Pointer
	_    unsafe.Pointer // bucket ptr at hash_iter initialization time
	_    unsafe.Pointer // current bucket
	_    unsafe.Pointer // keeps overflow buckets of hmap.buckets alive
	_    unsafe.Pointer // keeps overflow buckets of hmap.oldbuckets alive
	_    uintptr        // bucket iteration started at
	_    uint8          // intra-bucket offset to start from during iteration (should be big enough to hold bucketCnt-1)
	_    bool           // already wrapped around from end of bucket array to beginning
	_    uint8
	_    uint8
	_    uintptr
}

// StructType represents a content for struct types.
// it is used to get offsets/types of struct fields, see below.
type StructType struct {
	Type   Type
	_      [1]*byte
	Fields []StructField
}

// structField is a field in a struct.
// it contains its name, type, offset.
type StructField struct {
	Name   [1]*byte
	Type   *Type
	Offset uintptr
}

const (
	KindDirectIface = 1 << 5
	KindGCProg      = 1 << 6 // Type.gc points to GC program
	KindMask        = (1 << 5) - 1
)

// rtypeAndPointerOf is almost the same as reflect.ValueOf.
// see https://pkg.go.dev/reflect#ValueOf, but it doesn't
// have flags in it
func TypeAndPointerOf(this any) (*Type, unsafe.Pointer) {
	if this != nil {
		val := *(*struct {
			typ *Type
			ptr unsafe.Pointer
		})(unsafe.Pointer(&this))
		return val.typ, val.ptr
	}
	return nil, nil
}

// KeyAndElem returns type of a key and type of a elem.
// the former is nil when it's not a map,
// both are nil when it's not one of array, channel, pointer,
// slice, map.
func (typ *Type) KeyAndElem() (*Type, *Type) {
	switch reflect.Kind(typ.Kind & KindMask) {
	case reflect.Array, reflect.Chan, reflect.Pointer, reflect.Slice:
		return nil, (*struct {
			typ  Type
			elem *Type
		})(unsafe.Pointer(typ)).elem
	case reflect.Map:
		typs := (*struct {
			typ  Type
			key  *Type
			elem *Type
		})(unsafe.Pointer(typ))
		return typs.key, typs.elem
	}
	return nil, nil
}

// String converts byte slices to a string, using unsafe.
// this should be replaced after go releases new unsafe features.
func String(bytes []byte) string {
	return *(*string)(unsafe.Pointer(&bytes))
}

// TitleAndTag parses struct tag and field name in a struct.
// similar function is in the standard library.
func (structField StructField) TitleAndTag() (string, reflect.StructTag) {

	if structField.Name[0] == nil {
		return "", ""
	}

	one, two := structField.readVarint(1)
	title := *(*string)(unsafe.Pointer(&StringHeader{
		Ptr: unsafe.Add(unsafe.Pointer(structField.Name[0]), one+1),
		Len: two,
	}))

	if (*structField.Name[0])&(1<<1) == 0 {
		return title, ""
	}

	three, four := structField.readVarint(one + two + 1)
	tag := *(*reflect.StructTag)(unsafe.Pointer(&StringHeader{
		Ptr: unsafe.Add(unsafe.Pointer(structField.Name[0]), one+two+three+1),
		Len: four,
	}))
	// cast to reflect.StructTag, not string.
	// only difference is that you can call Get/Lookup method from it.
	return title, tag
}

// readVarint parses a varint as encoded by encoding/binary.
// It returns the number of encoded bytes and the encoded value.
func (structField StructField) readVarint(off int) (int, int) {
	val := 0
	for idx := 0; ; idx++ {
		u8 := *(*byte)(unsafe.Add(unsafe.Pointer(structField.Name[0]), off+idx))
		val += int(u8&0x7f) << (7 * idx)
		if u8&0x80 == 0 {
			return idx + 1, val
		}
	}
}
