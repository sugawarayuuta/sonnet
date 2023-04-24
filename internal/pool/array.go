package pool

import (
	"github.com/sugawarayuuta/sonnet/internal/types"
	"math/bits"
	"sync"
)

type ArrayPool struct {
	*types.Type
	shard [shard]sync.Pool
}

func (pool *ArrayPool) Get(size int) types.SliceHeader {
	idx := bits.Len(uint(size))
	if idx < shard && size != 0 {
		in := pool.shard[idx].Get()
		if in != nil {
			header := in.(*types.SliceHeader)
			if header.Cap >= size {
				return *header
			}
		}
	}
	return types.SliceHeader{
		Ptr: pool.NewArray(size),
		Cap: size,
	}
}

func (pool *ArrayPool) Put(header types.SliceHeader) {
	idx := bits.Len(uint(header.Cap))
	if idx >= shard {
		return
	}
	pool.shard[idx].Put(&header)
}
