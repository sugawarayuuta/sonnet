package mem

import (
	"math/bits"
	"sync"
)

var (
	buf   = newCirc[*sync.Pool](def)
	shift = 10
	mtx   sync.RWMutex
)

const (
	def = 10
)

func init() {
	// push out 5 nil sync.Pool's
	for idx := 0; idx < def; idx++ {
		var pool sync.Pool
		buf.append(&pool)
	}
}

func Get(capacity int) []byte {
	mtx.RLock()
	defer mtx.RUnlock()
	pos := bits.Len(uint(capacity)>>shift) - 1
	if uint(pos) < def {
		slice, ok := buf.at(pos).Get().(*[]byte)
		if ok && cap(*slice) >= capacity {
			return (*slice)[:capacity]
		}
	}
	return make([]byte, capacity)
}

func Put(slice []byte) {
	mtx.Lock()
	defer mtx.Unlock()
	// this -1 excludes 0's from the list.
	pos := bits.Len(uint(cap(slice))>>shift) - 1
	if pos < 0 && shift > 0 {
		var pool sync.Pool
		shift--
		pos++
		buf.prepend(&pool)
	} else if pos >= def {
		var pool sync.Pool
		shift++
		pos--
		buf.append(&pool)
	}
	if uint(pos) < def {
		buf.at(pos).Put(&slice)
	}
}
