package arith

import "math/bits"

const (
	x01 = 0x0101010101010101
)

func NextPow2(inp uint) uint {
	// https://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
	inp--
	inp |= inp >> 1
	inp |= inp >> 2
	inp |= inp >> 4
	inp |= inp >> 8
	inp |= inp >> 16
	inp |= inp >> 32
	inp++
	return inp
}

func EscapeHTML(u64 uint64) int {
	mask := u64 | (u64 - x01*' ') | (u64 ^ x01*'\\' - x01) | (u64 ^ x01*'"' - x01)
	mask |= (u64 ^ x01*'>' - x01) | (u64 ^ x01*'<' - x01) | (u64 ^ x01*'&' - x01)
	return bits.TrailingZeros64(mask&(x01*0x80)) >> 3
}

func Escape(u64 uint64) int {
	// https://graphics.stanford.edu/~seander/bithacks.html#HasLessInWord
	return bits.TrailingZeros64((u64|(u64-x01*' ')|(u64^x01*'\\'-x01)|(u64^x01*'"'-x01))&(x01*0x80)) >> 3
}

func NonSpace(u64 uint64) int {
	u64 |= (u64 ^ x01*' ' - x01) | (u64 ^ x01*'\n' - x01) | (u64 ^ x01*'\t' - x01) | (u64 ^ x01*'\r' - x01)
	return bits.TrailingZeros64(^u64) >> 3
}

func Uint64(u64 uint64) uint64 {
	const mask = 0x000000ff000000ff
	const fst = 100 + 1000000<<32
	const sec = 1 + 10000<<32
	u64 -= x01 * '0'
	u64 = u64*10 + u64>>8
	return (u64&mask*fst + u64>>16&mask*sec) >> 32
}

func CanUint64(u64 uint64) bool {
	return u64&(u64+x01*0x06)&(x01*0xf0) == x01*0x30
}