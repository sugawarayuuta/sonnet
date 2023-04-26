package decoder

import (
	"math/rand"
	"time"
)

type (
	table struct {
		pairs []pair
		mask  uint64
		seed  uint64
	}
	pair struct {
		field structField
		hash  uint64
	}
)

const (
	offset64 = uint64(14695981039346656037)
	prime64  = uint64(1099511628211)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (tab *table) prep(size uint64) {

	size--
	size |= size >> 1
	size |= size >> 2
	size |= size >> 4
	size |= size >> 8
	size |= size >> 16
	size |= size >> 32
	size++

	tab.pairs = make([]pair, size)
	tab.mask = size - 1
}

func (tab *table) set(strs []structField) {
cont:
	for idx := 0; idx < 100; idx++ {
		tab.seed = rand.Uint64()
		for _, str := range strs {
			hash := fnv(tab.seed, str.name)
			pos := hash & tab.mask
			if tab.pairs[pos].hash != 0 {
				tab.pairs = tab.pairs[:0]
				tab.pairs = tab.pairs[:tab.mask+1]
				continue cont
			}
			tab.pairs[pos] = pair{field: str, hash: hash}
		}
		return
	}
	tab.prep((tab.mask + 1) * 2)
	tab.set(strs)
}

func fnv(seed uint64, word string) uint64 {
	hash := offset64 ^ seed
	for len(word) >= 8 {
		hash = (hash ^ uint64(word[0])) * prime64
		hash = (hash ^ uint64(word[1])) * prime64
		hash = (hash ^ uint64(word[2])) * prime64
		hash = (hash ^ uint64(word[3])) * prime64
		hash = (hash ^ uint64(word[4])) * prime64
		hash = (hash ^ uint64(word[5])) * prime64
		hash = (hash ^ uint64(word[6])) * prime64
		hash = (hash ^ uint64(word[7])) * prime64
		word = word[8:]
	}
	if len(word) >= 4 {
		hash = (hash ^ uint64(word[0])) * prime64
		hash = (hash ^ uint64(word[1])) * prime64
		hash = (hash ^ uint64(word[2])) * prime64
		hash = (hash ^ uint64(word[3])) * prime64
		word = word[4:]
	}
	if len(word) >= 2 {
		hash = (hash ^ uint64(word[0])) * prime64
		hash = (hash ^ uint64(word[1])) * prime64
		word = word[2:]
	}
	if len(word) >= 1 {
		hash = (hash ^ uint64(word[0])) * prime64
	}
	return hash
}
