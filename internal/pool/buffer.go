package pool

import (
	"math/bits"
	"sync"
	"sync/atomic"
)

type (
	buffer struct {
		bytes []byte
	}
)

var (
	buffers [shard]sync.Pool
	iter    uintptr
)

const (
	shard = 16
	shift = 10
	Min   = 1 << shift
)

func Get(size int) []byte {
	div, rem := size>>shift, size&(Min-1)
	idx := bits.Len(uint(div))
	if div != 0 && rem == 0 {
		idx--
	}
	if idx < shard && size != 0 {
		in := buffers[idx].Get()
		if in != nil {
			bytes := in.(*buffer).bytes
			if cap(bytes) >= size {
				return bytes[:size]
			}
		}
	}
	return make([]byte, size)
}

func Iter() []byte {
	var uptr uintptr
	for ; uptr <= iter; uptr++ {
		in := buffers[iter-uptr].Get()
		if in == nil {
			continue
		}
		bytes := in.(*buffer).bytes
		if cap(bytes) >= Min {
			return bytes
		}
	}
	return make([]byte, 0, Min)
}

func Put(bytes []byte) {
	size := cap(bytes)
	div, rem := size>>shift, size&(Min-1)
	idx := bits.Len(uint(div))
	if div != 0 && rem == 0 {
		idx--
	}
	if idx >= shard {
		return
	}
	if uptr := uintptr(idx); uptr > iter {
		atomic.StoreUintptr(&iter, uptr)
	}
	buffers[idx].Put(&buffer{bytes: bytes[:0]})
}
