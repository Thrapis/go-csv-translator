package textutil

// CountLeadingSpaces returns the number of ASCII spaces at the start of s.
func CountLeadingSpaces(s string) int {
	count := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		count++
	}
	return count
}

// CountFinalSpaces returns the number of ASCII spaces at the end of s.
func CountFinalSpaces(s string) int {
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != ' ' {
			break
		}
		count++
	}
	return count
}
