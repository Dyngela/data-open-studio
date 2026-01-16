package lib

import (
	"fmt"
	"strings"
	"unicode"
)

// Concat joins multiple values with an optional separator
func Concat(separator string, values ...any) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		if v != nil {
			parts = append(parts, fmt.Sprintf("%v", v))
		}
	}
	return strings.Join(parts, separator)
}

// ConcatNoSep joins multiple values without separator
func ConcatNoSep(values ...any) string {
	return Concat("", values...)
}

// Upper converts value to uppercase string
func Upper(v any) string {
	if v == nil {
		return ""
	}
	return strings.ToUpper(fmt.Sprintf("%v", v))
}

// Lower converts value to lowercase string
func Lower(v any) string {
	if v == nil {
		return ""
	}
	return strings.ToLower(fmt.Sprintf("%v", v))
}

// Trim removes leading and trailing whitespace
func Trim(v any) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// TrimLeft removes leading whitespace
func TrimLeft(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	return strings.TrimLeftFunc(s, unicode.IsSpace)
}

// TrimRight removes trailing whitespace
func TrimRight(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// Left returns the first n characters
func Left(v any, n int) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	runes := []rune(s)
	if n > len(runes) {
		return s
	}
	if n < 0 {
		return ""
	}
	return string(runes[:n])
}

// Right returns the last n characters
func Right(v any, n int) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	runes := []rune(s)
	if n > len(runes) {
		return s
	}
	if n < 0 {
		return ""
	}
	return string(runes[len(runes)-n:])
}

// Substr returns a substring starting at index with given length
func Substr(v any, start, length int) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprintf("%v", v)
	runes := []rune(s)
	if start < 0 {
		start = 0
	}
	if start >= len(runes) {
		return ""
	}
	end := start + length
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[start:end])
}

// Replace replaces all occurrences of old with new
func Replace(v any, old, new string) string {
	if v == nil {
		return ""
	}
	return strings.ReplaceAll(fmt.Sprintf("%v", v), old, new)
}

// ReplaceFirst replaces the first occurrence of old with new
func ReplaceFirst(v any, old, new string) string {
	if v == nil {
		return ""
	}
	return strings.Replace(fmt.Sprintf("%v", v), old, new, 1)
}

// Contains checks if string contains substring
func Contains(v any, substr string) bool {
	if v == nil {
		return false
	}
	return strings.Contains(fmt.Sprintf("%v", v), substr)
}

// StartsWith checks if string starts with prefix
func StartsWith(v any, prefix string) bool {
	if v == nil {
		return false
	}
	return strings.HasPrefix(fmt.Sprintf("%v", v), prefix)
}

// EndsWith checks if string ends with suffix
func EndsWith(v any, suffix string) bool {
	if v == nil {
		return false
	}
	return strings.HasSuffix(fmt.Sprintf("%v", v), suffix)
}

// Length returns the length of the string
func Length(v any) int {
	if v == nil {
		return 0
	}
	return len([]rune(fmt.Sprintf("%v", v)))
}

// PadLeft pads string on the left to reach specified length
func PadLeft(v any, length int, pad string) string {
	if v == nil {
		v = ""
	}
	s := fmt.Sprintf("%v", v)
	if pad == "" {
		pad = " "
	}
	for len([]rune(s)) < length {
		s = pad + s
	}
	return Left(s, length)
}

// PadRight pads string on the right to reach specified length
func PadRight(v any, length int, pad string) string {
	if v == nil {
		v = ""
	}
	s := fmt.Sprintf("%v", v)
	if pad == "" {
		pad = " "
	}
	for len([]rune(s)) < length {
		s = s + pad
	}
	return Left(s, length)
}

// Split splits string by separator and returns slice
func Split(v any, separator string) []string {
	if v == nil {
		return []string{}
	}
	return strings.Split(fmt.Sprintf("%v", v), separator)
}

// ToString converts any value to string
func ToString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// Format formats a string with arguments (printf style)
func Format(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// Reverse reverses the string
func Reverse(v any) string {
	if v == nil {
		return ""
	}
	runes := []rune(fmt.Sprintf("%v", v))
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}