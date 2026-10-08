package textutil

import (
	"strings"
	"unicode/utf8"
)

// LimitUTF8Bytes removes invalid UTF-8 and returns at most maxBytes without
// splitting a multi-byte rune. It is intended for strings that cross a
// PostgreSQL or external-system boundary after being bounded for display.
func LimitUTF8Bytes(value string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	value = strings.ToValidUTF8(value, "")
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}
