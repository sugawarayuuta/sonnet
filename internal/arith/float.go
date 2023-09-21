package arith

import (
	"math"
	"math/bits"
)

var (
	// don't use math.Pow10, they don't use lookup tables directly,
	// and makes some operations which leads to inaccuracy.
	pow10 = [...]float64{
		1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11,
		1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22,
	}
)

// Lemire64 implements a 64 bit version of fast_double_parser,
// https://github.com/lemire/fast_double_parser/blob/master/include/fast_double_parser.h
// Which is used all over the place like Go, Rust, GCC.
func Lemire64(mant uint64, pow int64, neg bool) (float64, bool) {
	const (
		sign     = 1 << 63
		bias     = 1023
		lenFrac  = 53
		hiPow    = 308
		loPow    = -325
		log5log2 = 152170 + 1<<16
	)
	if absInt(pow) <= 22 && mant <= 1<<lenFrac-1 {
		f64 := float64(mant)
		if pow < 0 {
			f64 /= pow10[-pow]
		} else {
			f64 *= pow10[pow]
		}
		if neg {
			f64 = math.Float64frombits(math.Float64bits(f64) | sign)
		}
		return f64, true
	}
	if mant <= 0 {
		var u64 uint64
		if neg {
			u64 |= sign
		}
		return math.Float64frombits(u64), true
	}
	idx := pow - loPow
	if idx >= hiPow-(loPow-1) {
		return 0, false
	}
	exp := log5log2*pow>>16 + bias + 64
	lead := int64(bits.LeadingZeros64(mant))
	mant <<= lead
	hi64, lo64 := bits.Mul64(mant, mant64[idx])
	if hi64&0x1ff == 0x1ff && lo64+mant < lo64 {
		hi128, lo128 := bits.Mul64(mant, mant128[idx])
		mid := lo64 + hi128
		if mid < lo64 {
			hi64++
		}
		if hi64&0x1ff == 0x1ff && lo128+mant < lo128 && mid+1 == 0 {
			return 0, false
		}
		lo64 = mid
	}
	bit := hi64 >> 63
	ret := hi64 >> (bit + 9)
	lead += int64(bit ^ 1)
	if lo64 == 0 && hi64&0x1ff == 0 && ret&3 == 1 {
		return 0, false
	}
	ret = (ret + ret&1) >> 1
	if ret >= 1<<lenFrac {
		ret = 1 << 52
		lead--
	}
	ret &^= 1 << 52
	exp -= lead
	if exp < 1 || exp > 2046 {
		return 0, false
	}
	ret |= uint64(exp) << 52
	if neg {
		ret |= sign
	}
	return math.Float64frombits(ret), true
}

func absInt(i64 int64) int64 {
	mask := i64 >> (64 - 1)
	return (i64 + mask) ^ mask
}
