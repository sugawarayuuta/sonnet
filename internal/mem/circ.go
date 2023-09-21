package mem

import "github.com/sugawarayuuta/sonnet/internal/arith"

type (
	circ[typ any] struct {
		slice         []typ
		idx, len, cap uint
	}
)

func newCirc[typ any](length int) *circ[typ] {
	pow2 := arith.NextPow2(uint(length))
	return &circ[typ]{
		slice: make([]typ, pow2),
		len:   uint(length),
		cap:   pow2,
	}
}

func (buf *circ[typ]) append(val typ) {
	// zero-out (possibly nil-out) the empty element
	// so that it's easy for the GC
	var zero typ
	buf.slice[buf.idx] = zero
	buf.slice[(buf.idx+buf.len)&(buf.cap-1)] = val
	buf.idx = (buf.idx + 1) & (buf.cap - 1)
}

func (buf *circ[typ]) prepend(val typ) {
	var zero typ
	buf.idx = (buf.idx - 1) & (buf.cap - 1)
	buf.slice[(buf.idx+buf.len)&(buf.cap-1)] = zero
	buf.slice[buf.idx] = val
}

func (buf *circ[typ]) at(idx int) typ {
	return buf.slice[(buf.idx+uint(idx))&(buf.cap-1)]
}
