package decoder

import (
	"errors"
	"strconv"

	"github.com/sugawarayuuta/sonnet/internal/types"
)

func btoi(bytes []byte) (int64, error) {

	length := len(bytes)

	if length == 0 {
		return 0, errors.New("cannot create int64 from an empty string")
	}

	var pos int
	var i64 int64

	minus := bytes[0] == '-'
	if minus {
		pos++
	}

	if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
		i64 = i64*10 + int64(bytes[pos]-'0')
		pos++
		if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
			i64 = i64*10 + int64(bytes[pos]-'0')
			pos++
			if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
				i64 = i64*10 + int64(bytes[pos]-'0')
				pos++
				if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
					i64 = i64*10 + int64(bytes[pos]-'0')
					pos++
				}
			}
		}
	}

	for pos != length {
		if pos == 18 {
			return strconv.ParseInt(types.String(bytes), 10, 64)
		}
		if bytes[pos] >= '0' && bytes[pos] <= '9' {
			i64 = i64*10 + int64(bytes[pos]-'0')
			pos++
			continue
		}
		break
	}

	if pos != length {
		return 0, errors.New("failed to parse: " + string(bytes))
	}

	if minus {
		return -i64, nil
	}

	return i64, nil
}

func btou(bytes []byte) (uint64, error) {

	length := len(bytes)

	if length == 0 {
		return 0, errors.New("cannot create uint64 from an empty string")
	}

	var pos int
	var u64 uint64

	if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
		u64 = u64*10 + uint64(bytes[pos]-'0')
		pos++
		if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
			u64 = u64*10 + uint64(bytes[pos]-'0')
			pos++
			if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
				u64 = u64*10 + uint64(bytes[pos]-'0')
				pos++
				if pos != length && bytes[pos] >= '0' && bytes[pos] <= '9' {
					u64 = u64*10 + uint64(bytes[pos]-'0')
					pos++
				}
			}
		}
	}

	for pos != length {
		if pos == 18 {
			return strconv.ParseUint(types.String(bytes), 10, 64)
		}
		if bytes[pos] >= '0' && bytes[pos] <= '9' {
			u64 = u64*10 + uint64(bytes[pos]-'0')
			pos++
			continue
		}
		break
	}

	if pos != length {
		return 0, errors.New("failed to parse: " + string(bytes))
	}

	return u64, nil
}
