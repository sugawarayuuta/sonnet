package sonnet

import (
	"sync/atomic"
)

type (
	cache[key comparable, elm any] struct {
		ptr *atomic.Pointer[map[key]elm]
	}
)

func makeCache[key comparable, elm any]() *cache[key, elm] {
	var atom atomic.Pointer[map[key]elm]
	mapper := make(map[key]elm)
	atom.Store(&mapper)
	return &cache[key, elm]{ptr: &atom}
}

func (cac *cache[key, elm]) get(ref key) (elm, bool) {
	mapper := *cac.ptr.Load()
	dec, ok := mapper[ref]
	return dec, ok
}

func (cac *cache[key, elm]) set(ref key, val elm) {
	mapper := *cac.ptr.Load()
	rep := make(map[key]elm, len(mapper)+1)
	for key, elm := range mapper {
		rep[key] = elm
	}
	rep[ref] = val
	cac.ptr.Store(&rep)
}
