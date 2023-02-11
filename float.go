package sonnet

import (
	"math"
)

func f64(bytes []byte) float64 {

	if len(bytes) == 0 {
		return 0
	}

	pos := 0
	minus := bytes[0] == '-'
	if minus {
		pos++
	}
	var u64 uint64

	for pos < len(bytes) {
		if bytes[pos] >= '0' && bytes[pos] <= '9' {
			u64 = u64*10 + uint64(bytes[pos]-'0')
			pos++
		} else {
			break
		}
	}

	f64 := float64(u64)
	if pos == len(bytes) {
		if minus {
			return -f64
		}
		return f64
	}

	if bytes[pos] == '.' {
		pos++
		if pos == len(bytes) {
			return 0
		}
		prev := pos

		for pos < len(bytes) {
			if bytes[pos] >= '0' && bytes[pos] <= '9' {
				u64 = u64*10 + uint64(bytes[pos]-'0')
				pos++
			} else {
				break
			}
		}
		f64 = float64(u64) / math.Pow10(pos-prev)
		if pos == len(bytes) {
			if minus {
				return -f64
			}
			return f64
		}

	}

	if bytes[pos] == 'E' || bytes[pos] == 'e' {
		pos++
		if pos == len(bytes) {
			return 0
		}
		var eminus bool
		if bytes[pos] == '+' || bytes[pos] == '-' {
			eminus = bytes[pos] == '-'
			pos++
			if pos == len(bytes) {
				return 0
			}
		}
		var e int

		for pos < len(bytes) {
			if bytes[pos] >= '0' && bytes[pos] <= '9' {
				e = e*10 + int(bytes[pos]-'0')
				pos++
			} else {
				break
			}
		}
		if eminus {
			e = -e
		}
		f64 *= math.Pow10(e)
		if pos == len(bytes) {
			if minus {
				return -f64
			}
			return f64
		}
	}

	return 0
}

func i64(bytes []byte) int64 {

	if len(bytes) == 0 {
		return 0
	}

	pos := 0
	minus := bytes[0] == '-'
	if minus {
		pos++
	}
	var i64 int64

	for pos < len(bytes) {
		if bytes[pos] >= '0' && bytes[pos] <= '9' {
			i64 = i64*10 + int64(bytes[pos]-'0')
			pos++
		} else {
			break
		}
	}

	if pos == len(bytes) {
		if minus {
			return -i64
		}
		return i64
	}

	return 0
}

func u64(bytes []byte) uint64 {
	if len(bytes) == 0 {
		return 0
	}

	pos := 0
	var u64 uint64

	for pos < len(bytes) {
		if bytes[pos] >= '0' && bytes[pos] <= '9' {
			u64 = u64*10 + uint64(bytes[pos]-'0')
			pos++
		} else {
			break
		}
	}

	if pos == len(bytes) {
		return u64
	}

	return 0
}
