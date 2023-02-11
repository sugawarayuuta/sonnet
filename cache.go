package sonnet

import (
	"sync"
	"unsafe"
)

type cache struct {
	functions map[*rtype]function
	lock      sync.Mutex
}

var (
	global = func() [caches]*cache {
		var caches [caches]*cache
		for idx := range caches {
			caches[idx] = &cache{functions: make(map[*rtype]function, 4)}
		}
		return caches
	}()
)

const (
	caches = 12
)

func load(typ *rtype) (function, bool) {
	do, ok := global[uintptr(unsafe.Pointer(typ))%caches].functions[typ]
	return do, ok
}

func save(typ *rtype, do function) {
	cache := global[uintptr(unsafe.Pointer(typ))%caches]
	cache.lock.Lock()
	cache.functions[typ] = do
	cache.lock.Unlock()
}
