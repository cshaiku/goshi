package detect

import (
	"strings"
	"unicode"
)

// Tokenize normalizes and splits input into word tokens.
// Only letters and numbers are kept; everything else is a separator.
func Tokenize(input string) []string {
	input = strings.ToLower(input)

	var b strings.Builder
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}

	return strings.Fields(b.String())
}
