package decoder

import (
	"sync/atomic"
	"unsafe"

	"github.com/sugawarayuuta/sonnet/internal/types"
)

var (
	global [shard]unsafe.Pointer
)

const (
	shard = 16
)

func init() {
	for idx := range global {
		cache := make(map[*types.Type]codec)
		global[idx] = *(*unsafe.Pointer)(unsafe.Pointer(&cache))
	}
}

func load(typ *types.Type) (codec, bool) {
	uptr := uintptr(unsafe.Pointer(typ)) & (shard - 1)
	ptr := atomic.LoadPointer(&global[uptr])
	cache := *(*map[*types.Type]codec)(unsafe.Pointer(&ptr))
	fun, ok := cache[typ]
	return fun, ok
}

func store(typ *types.Type, fun codec) {
	uptr := uintptr(unsafe.Pointer(typ)) & (shard - 1)
	for {
		ptr := atomic.LoadPointer(&global[uptr])
		cache := *(*map[*types.Type]codec)(unsafe.Pointer(&ptr))
		mapper := make(map[*types.Type]codec, len(cache)+1)
		for key, val := range cache {
			mapper[key] = val
		}
		mapper[typ] = fun
		if atomic.CompareAndSwapPointer(&global[uptr], ptr, *(*unsafe.Pointer)(unsafe.Pointer(&mapper))) {
			return
		}
	}
}
