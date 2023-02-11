package sonnet

import (
	"reflect"
	"unsafe"
)

//go:linkname (*rtype).unsafe_NewArray reflect.unsafe_NewArray
//go:noescape
func (*rtype) unsafe_NewArray(int) unsafe.Pointer

//go:linkname (*rtype).typedslicecopy reflect.typedslicecopy
//go:noescape
func (*rtype) typedslicecopy(sliceHeader, sliceHeader) int

//go:linkname (*rtype).std reflect.toType
//go:noescape
func (*rtype) std() reflect.Type

//go:linkname (*rtype).mapassign_faststr runtime.mapassign_faststr
//go:noescape
func (*rtype) mapassign_faststr(unsafe.Pointer, string) unsafe.Pointer
