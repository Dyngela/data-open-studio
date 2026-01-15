package lib

import "strings"

func Concat(separator string, str ...string) string {
	return strings.Join(str, separator)
}
