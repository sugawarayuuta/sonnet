package encoder

import (
	"math"
	"reflect"
	"strconv"

	"github.com/sugawarayuuta/sonnet/internal/compat"
)

const (
	maxUint64Length = 20
)

func utob(u64 uint64) []byte {
	pos := maxUint64Length
	var dst [maxUint64Length]byte
	for {
		pos--
		dst[pos], u64 = '0'+byte(u64%10), u64/10
		if u64 == 0 {
			return dst[pos:]
		}
	}
}

func itob(i64 int64) []byte {
	sign := i64 < 0
	if sign {
		i64 = -i64
	}
	pos := maxUint64Length
	var dst [maxUint64Length]byte
	for {
		pos--
		dst[pos], i64 = '0'+byte(i64%10), i64/10
		if i64 == 0 {
			if sign {
				pos--
				dst[pos] = '-'
			}
			return dst[pos:]
		}
	}
}

func ftob(dst []byte, f64 float64, bits int) ([]byte, error) {
	if math.IsInf(f64, 0) || math.IsNaN(f64) {
		return nil, compat.NewUnsupportedValueError(reflect.ValueOf(f64), strconv.FormatFloat(f64, 'g', -1, bits))
	}
	abs := math.Abs(f64)
	fmt := byte('f')
	// Note: Must use float32 comparisons for underlying float32 value to get precise cutoffs right.
	if abs != 0 {
		if bits == 64 && (abs < 1e-6 || abs >= 1e21) || bits == 32 && (float32(abs) < 1e-6 || float32(abs) >= 1e21) {
			fmt = 'e'
		}
	}
	dst = strconv.AppendFloat(dst, f64, fmt, -1, bits)
	if fmt == 'e' {
		// clean up e-09 to e-9
		n := len(dst)
		if n >= 4 && dst[n-4] == 'e' && dst[n-3] == '-' && dst[n-2] == '0' {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
	}
	return dst, nil
}
