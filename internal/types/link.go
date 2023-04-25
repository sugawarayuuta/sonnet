package types

import (
	"reflect"
	"unsafe"
)

//go:linkname unsafe_New reflect.unsafe_New
//go:noescape
func unsafe_New(*Type) unsafe.Pointer

// New returns a new unsafe.Pointer of a value
func (typ *Type) New() unsafe.Pointer {
	return unsafe_New(typ)
}

//go:linkname unsafe_NewArray reflect.unsafe_NewArray
//go:noescape
func unsafe_NewArray(*Type, int) unsafe.Pointer

// NewArray creates a new array. it is used when slices
// get resized. the parameter is the capacity.
func (typ *Type) NewArray(capacity int) unsafe.Pointer {
	return unsafe_NewArray(typ, capacity)
}

//go:linkname typedslicecopy reflect.typedslicecopy
//go:noescape
func typedslicecopy(*Type, SliceHeader, SliceHeader) int

// TypedSliceCopy copies slices of a specific type.
// it is used when slices get resized.
func (typ *Type) TypedSliceCopy(dst SliceHeader, src SliceHeader) int {
	return typedslicecopy(typ, dst, src)
}

//go:linkname toType reflect.toType
//go:noescape
func toType(*Type) reflect.Type

// Reflect is a function that converts *Type into reflect.Type
// to keep compatibility in errors.
func (typ *Type) Reflect() reflect.Type {
	return toType(typ)
}

//go:linkname mapassign_faststr runtime.mapassign_faststr
//go:noescape
func mapassign_faststr(*Type, unsafe.Pointer, string) unsafe.Pointer

// MapAssignFaststr creates a new space in the Go map structure.
func (typ *Type) MapAssignFaststr(ptr unsafe.Pointer, key string) unsafe.Pointer {
	return mapassign_faststr(typ, ptr, key)
}

//go:linkname mapassign runtime.mapassign
//go:noescape
func mapassign(*Type, unsafe.Pointer, unsafe.Pointer) unsafe.Pointer

// MapAssign creates a new space in the Go map structure.
// it is used if the element is too big, or the element is not a string.
func (typ *Type) MapAssign(ptr unsafe.Pointer, key unsafe.Pointer) unsafe.Pointer {
	return mapassign(typ, ptr, key)
}

//go:noescape
//go:linkname mapiterinit runtime.mapiterinit
func mapiterinit(*Type, unsafe.Pointer, *Iter)

// MapIterInit initializes map iterators.
func (typ *Type) MapIterInit(ptr unsafe.Pointer, iter *Iter) {
	mapiterinit(typ, ptr, iter)
}

// MapIterNext change key/elem property to the next set in the map.
//
//go:noescape
//go:linkname MapIterNext reflect.mapiternext
func MapIterNext(*Iter)

// MallocGC allocates. it's used when you want to avoid initializing bytes
//
//go:noescape
//go:linkname MallocGC runtime.mallocgc
func MallocGC(uintptr, *Type, bool) unsafe.Pointer
