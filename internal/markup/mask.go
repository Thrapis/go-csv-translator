package markup

import (
	"regexp"
	"strconv"
	"strings"
)

var maskRe = regexp.MustCompile(`§(\d+)§`)

// StripSentinels removes every §i§ sentinel from s.
func StripSentinels(s string) string {
	return maskRe.ReplaceAllString(s, "")
}

// SentinelsIntact reports whether s carries exactly the sentinels §0§ … §(n-1)§,
// each once, with no other "§" debris. A translator that mangled a sentinel
// (dropped a digit, duplicated a "§") fails this check, and the caller should
// fall back to fragment-by-fragment translation.
func SentinelsIntact(s string, n int) bool {
	seen := make([]bool, n)
	matches := maskRe.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		i, err := strconv.Atoi(m[1])
		if err != nil || i < 0 || i >= n || seen[i] {
			return false
		}
		seen[i] = true
	}
	for _, ok := range seen {
		if !ok {
			return false
		}
	}
	return strings.Count(s, "§") == 2*n
}

// Unmask replaces each §i§ sentinel in s with markers[i]. Markers whose sentinel
// the translator dropped are appended to the end (better than losing the
// markup); an unknown index is left in place.
func Unmask(s string, markers []string) string {
	seen := make([]bool, len(markers))
	seenCount := 0

	out := maskRe.ReplaceAllStringFunc(s, func(m string) string {
		i, err := strconv.Atoi(m[len("§") : len(m)-len("§")])
		if err != nil || i < 0 || i >= len(markers) {
			return m
		}
		if !seen[i] {
			seen[i], seenCount = true, seenCount+1
		}
		return markers[i]
	})

	// Only recover dropped markers when the translator kept at least one
	// sentinel; a string with none was probably never masked.
	if seenCount == 0 || seenCount == len(markers) {
		return out
	}
	var missing strings.Builder
	for i, ok := range seen {
		if !ok {
			missing.WriteString(markers[i])
		}
	}
	return out + missing.String()
}
