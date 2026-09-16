package domain

import "strings"

// TrimText applies the stable storage rule for user-entered business text.
// It deliberately does not change case because the original value is shown to
// users.
func TrimText(value string) string {
	return strings.TrimSpace(value)
}

// NormalizeSearchText applies the stable index and lookup rule for business
// names and locations. Display text remains trimmed but otherwise unchanged.
func NormalizeSearchText(value string) string {
	return strings.ToLower(TrimText(value))
}

// TrimOptionalText returns a trimmed copy, preserving nil to distinguish an
// omitted update from an explicit empty value.
func TrimOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := TrimText(*value)
	return &trimmed
}

// NormalizeOptionalSearchText returns a normalized copy for nullable indexed
// fields, preserving nil.
func NormalizeOptionalSearchText(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := NormalizeSearchText(*value)
	return &normalized
}
