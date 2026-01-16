package lib

import (
	"fmt"
	"math"
	"strconv"
)

// toFloat64 converts any numeric value to float64
func toFloat64(v any) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	case string:
		f, _ := strconv.ParseFloat(n, 64)
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return f
	}
}

// Add returns sum of two values
func Add(a, b any) float64 {
	return toFloat64(a) + toFloat64(b)
}

// Sub returns difference of two values
func Sub(a, b any) float64 {
	return toFloat64(a) - toFloat64(b)
}

// Mul returns product of two values
func Mul(a, b any) float64 {
	return toFloat64(a) * toFloat64(b)
}

// Div returns division of two values (returns 0 if divisor is 0)
func Div(a, b any) float64 {
	divisor := toFloat64(b)
	if divisor == 0 {
		return 0
	}
	return toFloat64(a) / divisor
}

// Mod returns modulo of two values
func Mod(a, b any) float64 {
	return math.Mod(toFloat64(a), toFloat64(b))
}

// Abs returns absolute value
func Abs(v any) float64 {
	return math.Abs(toFloat64(v))
}

// Round rounds to specified decimal places
func Round(v any, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(toFloat64(v)*multiplier) / multiplier
}

// Floor rounds down to nearest integer
func Floor(v any) float64 {
	return math.Floor(toFloat64(v))
}

// Ceil rounds up to nearest integer
func Ceil(v any) float64 {
	return math.Ceil(toFloat64(v))
}

// Pow returns a raised to power b
func Pow(a, b any) float64 {
	return math.Pow(toFloat64(a), toFloat64(b))
}

// Sqrt returns square root
func Sqrt(v any) float64 {
	return math.Sqrt(toFloat64(v))
}

// Min returns the minimum of two values
func Min(a, b any) float64 {
	return math.Min(toFloat64(a), toFloat64(b))
}

// Max returns the maximum of two values
func Max(a, b any) float64 {
	return math.Max(toFloat64(a), toFloat64(b))
}

// MinOf returns the minimum of multiple values
func MinOf(values ...any) float64 {
	if len(values) == 0 {
		return 0
	}
	min := toFloat64(values[0])
	for _, v := range values[1:] {
		if f := toFloat64(v); f < min {
			min = f
		}
	}
	return min
}

// MaxOf returns the maximum of multiple values
func MaxOf(values ...any) float64 {
	if len(values) == 0 {
		return 0
	}
	max := toFloat64(values[0])
	for _, v := range values[1:] {
		if f := toFloat64(v); f > max {
			max = f
		}
	}
	return max
}

// ToInt converts value to int64
func ToInt(v any) int64 {
	return int64(toFloat64(v))
}

// ToFloat converts value to float64
func ToFloat(v any) float64 {
	return toFloat64(v)
}

// Sign returns -1, 0, or 1 depending on sign of value
func Sign(v any) int {
	f := toFloat64(v)
	if f < 0 {
		return -1
	}
	if f > 0 {
		return 1
	}
	return 0
}

// IsZero returns true if value is zero
func IsZero(v any) bool {
	return toFloat64(v) == 0
}

// Clamp restricts value to range [min, max]
func Clamp(v, minVal, maxVal any) float64 {
	f := toFloat64(v)
	min := toFloat64(minVal)
	max := toFloat64(maxVal)
	if f < min {
		return min
	}
	if f > max {
		return max
	}
	return f
}

// Percent calculates percentage: (value / total) * 100
func Percent(value, total any) float64 {
	t := toFloat64(total)
	if t == 0 {
		return 0
	}
	return (toFloat64(value) / t) * 100
}

// PercentOf calculates percent of value: value * (percent / 100)
func PercentOf(value, percent any) float64 {
	return toFloat64(value) * (toFloat64(percent) / 100)
}