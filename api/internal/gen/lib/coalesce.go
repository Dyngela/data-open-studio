package lib

// Coalesce returns the first non-nil value
func Coalesce(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// IfNull returns defaultVal if v is nil, otherwise returns v
func IfNull(v, defaultVal any) any {
	if v == nil {
		return defaultVal
	}
	return v
}

// NullIf returns nil if v equals compareVal, otherwise returns v
func NullIf(v, compareVal any) any {
	if v == compareVal {
		return nil
	}
	return v
}

// IsNull returns true if v is nil
func IsNull(v any) bool {
	return v == nil
}

// IsNotNull returns true if v is not nil
func IsNotNull(v any) bool {
	return v != nil
}

// If returns trueVal if condition is true, otherwise falseVal
func If(condition bool, trueVal, falseVal any) any {
	if condition {
		return trueVal
	}
	return falseVal
}

// IfExpr returns trueVal if v is truthy (not nil, not zero, not empty string)
func IfExpr(v any, trueVal, falseVal any) any {
	if IsTruthy(v) {
		return trueVal
	}
	return falseVal
}

// IsTruthy returns true if value is considered "truthy"
func IsTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case int8:
		return val != 0
	case int16:
		return val != 0
	case int32:
		return val != 0
	case int64:
		return val != 0
	case uint:
		return val != 0
	case uint8:
		return val != 0
	case uint16:
		return val != 0
	case uint32:
		return val != 0
	case uint64:
		return val != 0
	case float32:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != ""
	default:
		return true
	}
}

// IsFalsy returns true if value is considered "falsy"
func IsFalsy(v any) bool {
	return !IsTruthy(v)
}