package types

import (
	"reflect"
	"unsafe"
)

// New returns a new unsafe.Pointer of a value
//
//go:linkname (*Type).New reflect.unsafe_New
//go:noescape
func (*Type) New() unsafe.Pointer

// NewArray creates a new array. it is used when slices
// get resized. the parameter is the capacity.
//
//go:linkname (*Type).NewArray reflect.unsafe_NewArray
//go:noescape
func (*Type) NewArray(int) unsafe.Pointer

// TypedSliceCopy copies slices of a specific type.
// it is used when slices get resized.
//
//go:linkname (*Type).TypedSliceCopy reflect.typedslicecopy
//go:noescape
func (*Type) TypedSliceCopy(SliceHeader, SliceHeader) int

// Reflect is a function that converts *Type into reflect.Type
// to keep compatibility in errors.
//
//go:linkname (*Type).Reflect reflect.toType
//go:noescape
func (*Type) Reflect() reflect.Type

// MapAssignFaststr creates a new space in the Go map structure.
//
//go:linkname (*Type).MapAssignFaststr runtime.mapassign_faststr
//go:noescape
func (*Type) MapAssignFaststr(unsafe.Pointer, string) unsafe.Pointer

// MapAssign creates a new space in the Go map structure.
// it is used if the element is too big, or the element is not a string.
//
//go:linkname (*Type).MapAssign runtime.mapassign
//go:noescape
func (*Type) MapAssign(unsafe.Pointer, unsafe.Pointer) unsafe.Pointer

// MapIterInit initializes map iterators.
//
//go:noescape
//go:linkname (*Type).MapIterInit runtime.mapiterinit
func (*Type) MapIterInit(unsafe.Pointer, *Iter)

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
