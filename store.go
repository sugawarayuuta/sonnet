package sonnet

import "hash/maphash"

type store struct {
	cap     uint64
	len     uint64
	mask    uint64
	buckets []bucket
}

type bucket struct {
	dib uint64
	key string
	val structField
}

const (
	maxHash = ^uint64(0) >> 16
	maxDib  = ^uint64(0) >> 48
)

var (
	seed = maphash.MakeSeed()
)

func (str *store) init(cap uint64) {
	str.cap = cap
	len := uint64(8)
	for len < cap {
		len *= 2
	}
	str.buckets = make([]bucket, len)
	str.mask = len - 1
}

func (str *store) get(bytes []byte) (structField, bool) {

	hash := maphash.Bytes(seed, bytes) >> 16
	idx := hash & str.mask
	for {
		bucket := str.buckets[idx]

		if bucket.dib&maxDib == 0 {
			return structField{}, false
		}

		if bucket.dib>>16 == hash {
			return bucket.val, true
		}

		idx = (idx + 1) & str.mask
	}
}

func (str *store) set(key string, val structField) {

	dib := maphash.String(seed, key)
	incoming := bucket{dib: dib, key: key, val: val}
	idx := (dib >> 16) & str.mask
	for {
		bucket := str.buckets[idx]

		if bucket.dib&maxDib == 0 {
			str.buckets[idx] = incoming
			str.len++
			return
		}

		if incoming.dib == bucket.dib {
			str.buckets[idx] = incoming
			return
		}

		if bucket.dib < incoming.dib {
			str.buckets[idx], incoming = incoming, bucket
		}

		idx = (idx + 1) & str.mask

		incoming.dib = incoming.dib | ((incoming.dib>>16)+1)&maxDib
	}
}
