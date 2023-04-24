package decoder

import (
	"github.com/sugawarayuuta/sonnet/internal/types"
	"math/bits"
	"unicode"
	"unsafe"
)

var (
	hex = [256]int{
		'0': 0,
		'1': 1,
		'2': 2,
		'3': 3,
		'4': 4,
		'5': 5,
		'6': 6,
		'7': 7,
		'8': 8,
		'9': 9,
		'A': 10,
		'B': 11,
		'C': 12,
		'D': 13,
		'E': 14,
		'F': 15,
		'a': 10,
		'b': 11,
		'c': 12,
		'd': 13,
		'e': 14,
		'f': 15,
	}
	unescapeTab = [256]bool{
		'\\': true,
		'"':  true,
	}
	fallbackIndex func([]byte) int
)

const (
	hi uint64 = 0x8080808080808080
	lo uint64 = 0x0101010101010101
)

const (
	backslash uint64 = lo * '\\'
	quote     uint64 = lo * '"'
	control   uint64 = lo * ' '
)

func init() {
	for char := byte(unicode.MaxASCII + 1); char != ' '; char++ {
		unescapeTab[char] = true
	}
	var double [2]byte
	*(*uint16)(unsafe.Pointer(&double)) = uint16(0xABCD)
	if double == [2]byte{0xAB, 0xCD} {
		fallbackIndex = fallbackIndexBig
	} else {
		fallbackIndex = fallbackIndexLittle
	}
}

func utor(src []byte) rune {
	var run rune
	for idx := range src {
		run = run*16 + rune(hex[src[idx]])
	}
	return run
}

func fallbackIndexLittle(src []byte) int {
	header := *(*types.SliceHeader)(unsafe.Pointer(&src))
	for header.Len >= 8 {
		u64 := *(*uint64)(header.Ptr)
		mask := u64 | (u64 - control) | (u64 ^ backslash - lo) | (u64 ^ quote - lo)
		if and := mask & hi; and != 0 {
			return len(src) - header.Len + int(((and-1)&lo*lo)>>(64-8)-1)
		}
		header.Ptr = unsafe.Add(header.Ptr, 8)
		header.Len -= 8
	}
	for header.Len >= 1 {
		if unescapeTab[*(*uint8)(header.Ptr)] {
			return len(src) - header.Len
		}
		header.Ptr = unsafe.Add(header.Ptr, 1)
		header.Len -= 1
	}
	return -1
}

func fallbackIndexBig(src []byte) int {
	header := *(*types.SliceHeader)(unsafe.Pointer(&src))
	for header.Len >= 8 {
		u64 := bits.ReverseBytes64(*(*uint64)(header.Ptr))
		mask := u64 | (u64 - control) | (u64 ^ backslash - lo) | (u64 ^ quote - lo)
		if and := mask & hi; and != 0 {
			return len(src) - header.Len + int(((and-1)&lo*lo)>>(64-8)-1)
		}
		header.Ptr = unsafe.Add(header.Ptr, 8)
		header.Len -= 8
	}
	for header.Len >= 1 {
		if unescapeTab[*(*uint8)(header.Ptr)] {
			return len(src) - header.Len
		}
		header.Ptr = unsafe.Add(header.Ptr, 1)
		header.Len -= 1
	}
	return -1
}
