package arith

import "math/bits"

const (
	x01 = 0x0101010101010101
)

func NextPow2(inp uint) uint {
	// https://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
	// discussion: it might be better to just 1<<bits.Len(inp) ?
	// it's not like this is used heavily anyway, not the first thing to do.
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
	// reduced instructions using 2 techniques used below - see Escape and NonSpace.
	mask := u64 | (u64 ^ x01*0x02 - x01*'!') | (u64 ^ x01*'\\' - x01)
	mask |= (u64 | x01*0x02 ^ x01*'>' - x01) | (u64 ^ x01*'&' - x01)
	return bits.TrailingZeros64(mask&(x01*0x80)) >> 3
}

func Escape(u64 uint64) int {
	// the XOR x01*0x02 part flips high 2 bytes and low 2 bytes of a block with 4 bytes in ASCII table.
	// which is exactly what we want. '!' is just 0x20 + 1. see the table of char^0x02-'!' below.
	// ...
	// '\x1a',  26: 00011010 -> '\x1a',  26: 11110111
	// '\x1b',  27: 00011011 -> '\x1b',  27: 11111000
	// '\x1c',  28: 00011100 -> '\x1c',  28: 11111101
	// '\x1d',  29: 00011101 -> '\x1d',  29: 11111110
	// '\x1e',  30: 00011110 -> '\x1e',  30: 11111011
	// '\x1f',  31: 00011111 -> '\x1f',  31: 11111100
	//    ' ',  32: 00100000 ->    ' ',  32: 00000001
	//    '!',  33: 00100001 ->    '!',  33: 00000010
	//    '"',  34: 00100010 ->    '"',  34: 11111111
	//    '#',  35: 00100011 ->    '#',  35: 00000000
	// ...
	// the bytes with their highest bits set are reported.
	return bits.TrailingZeros64((u64|(u64^x01*0x02-x01*'!')|(u64^x01*'\\'-x01))&(x01*0x80)) >> 3
}

func NonSpace(u64 uint64) int {
	// the OR x01*0x04 part make \t characters \r, because they're exactly 1 bit apart.
	// then, we can use the SWAR algorithm: https://graphics.stanford.edu/~seander/bithacks.html#HasLessInWord
	// hasless(word, 1) should work well here after XORing the \r at each byte.
	u64 |= (u64 ^ x01*' ' - x01) | (u64 ^ x01*'\n' - x01) | (u64 | x01*0x04 ^ x01*'\r' - x01)
	return bits.TrailingZeros64(^u64) >> 3
}

func Uint64(u64 uint64) uint64 {
	// SWAR-based integer parting algorithm described below:
	// http://govnokod.ru/13461#comment189156
	u64 = u64 & 0x0F0F0F0F0F0F0F0F * 2561 >> 8
	u64 = u64 & 0x00FF00FF00FF00FF * 6553601 >> 16
	u64 = u64 & 0x0000FFFF0000FFFF * 42949672960001 >> 32
	return u64
}

func CanUint64(u64 uint64) bool {
	// a branchy version might be faster, it says. but I couldn't see any speed gains from using branches
	// even if branches look predictable.
	// https://lemire.me/blog/2018/09/30/quickly-identifying-a-sequence-of-digits-in-a-string-of-characters/
	return u64&(u64+x01*0x06)&(x01*0xf0) == x01*0x30
}
