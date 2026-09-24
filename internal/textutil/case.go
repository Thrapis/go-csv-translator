// Package textutil holds small string helpers shared by the pipeline and games.
package textutil

import "unicode"

// IsUpper reports whether r is an uppercase letter.
func IsUpper(r rune) bool {
	return unicode.IsLetter(r) && unicode.IsUpper(r)
}

// IsLower reports whether r is a lowercase letter.
func IsLower(r rune) bool {
	return unicode.IsLetter(r) && unicode.IsLower(r)
}
