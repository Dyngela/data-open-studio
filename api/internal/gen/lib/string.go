package lib

import (
	"fmt"
	"strings"
)

// Concat joins values with a separator. Accepts any type — non-string values
// are converted via fmt.Sprintf.
func Concat(sep string, values ...any) string {
	parts := make([]string, len(values))
	for i, v := range values {
		switch s := v.(type) {
		case string:
			parts[i] = s
		default:
			parts[i] = fmt.Sprintf("%v", s)
		}
	}
	return strings.Join(parts, sep)
}
