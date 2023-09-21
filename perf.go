package sonnet

import (
	"github.com/sugawarayuuta/sonnet/internal/arith"
	"math/rand"
)

type (
	perf struct {
		tups []tuple
		mask uint32
		seed uint32
	}
	tuple struct {
		key  []byte
		elm  *field
		hash uint32
	}
)

const (
	wy1 = 0x53c5ca59
	wy2 = 0x74743c1b
	wy3 = wy1 | wy2<<32
)

const (
	lo8  uint8  = 0x20
	lo32 uint32 = 0x20202020
	lo64 uint64 = 0x2020202020202020
)

func makePerf(size int) *perf {
	var prf perf
	if size <= 0 {
		size = 1
	}
	prf.tups = make([]tuple, arith.NextPow2(uint(size)))
	prf.mask = uint32(len(prf.tups)) - 1
	return &prf
}

func (prf *perf) set(tups []tuple) {
cont:
	for idx := 0; idx < 500; idx++ {
		for idx := range prf.tups {
			prf.tups[idx] = tuple{}
		}
		prf.seed = rand.Uint32()
		for _, tup := range tups {
			hash := hash32(tup.key, prf.seed)
			for hash == 0 {
				continue cont
			}
			pos := hash & prf.mask
			if prf.tups[pos].hash != 0 {
				continue cont
			}
			tup.hash = hash
			prf.tups[pos] = tup
		}
		return
	}
	*prf = *makePerf(len(prf.tups) << 1)
	prf.set(tups)
}

func hash32(buf []byte, seed uint32) uint32 {
	hash := uint64(seed^wy1) * uint64(len(buf)^wy2)
	if len(buf) != 0 {
		for ; len(buf) > 8; buf = buf[8:] {
			hash = hash ^ lit.Uint64(buf) ^ wy3
			hash = (hash >> 32) * (hash & 0xffffffff)
		}
		hi, lo := hash>>32, hash&0xffffffff
		if len(buf) >= 4 {
			hi ^= uint64(lit.Uint32(buf))
			lo ^= uint64(lit.Uint32(buf[len(buf)-4:]))
		} else {
			lo ^= uint64(buf[0])
			lo ^= uint64(buf[len(buf)>>1]) << 8
			lo ^= uint64(buf[len(buf)-1]) << 16
		}
		hash = (lo^wy1)*(hi^wy2) ^ wy3
		hash = (hash >> 32) * (hash & 0xffffffff)
	}
	return uint32(hash>>32) ^ uint32(hash)
}

func hash32Up(buf []byte, seed uint32) uint32 {
	hash := uint64(seed^wy1) * uint64(len(buf)^wy2)
	if len(buf) != 0 {
		for ; len(buf) > 8; buf = buf[8:] {
			hash = hash ^ lit.Uint64(buf)&^lo64 ^ wy3
			hash = (hash >> 32) * (hash & 0xffffffff)
		}
		hi, lo := hash>>32, hash&0xffffffff
		if len(buf) >= 4 {
			hi ^= uint64(lit.Uint32(buf) &^ lo32)
			lo ^= uint64(lit.Uint32(buf[len(buf)-4:]) &^ lo32)
		} else {
			lo ^= uint64(buf[0] &^ lo8)
			lo ^= uint64(buf[len(buf)>>1]&^lo8) << 8
			lo ^= uint64(buf[len(buf)-1]&^lo8) << 16
		}
		hash = (lo^wy1)*(hi^wy2) ^ wy3
		hash = (hash >> 32) * (hash & 0xffffffff)
	}
	return uint32(hash>>32) ^ uint32(hash)
}
