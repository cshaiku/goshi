package detect

import "strings"

// IsLikelyFSList returns true if the utterance refers to a directory listing.
func IsLikelyFSList(input string) bool {
	s := strings.ToLower(input)

	return strings.Contains(s, "list") ||
		strings.Contains(s, "files") ||
		strings.Contains(s, "folder") ||
		strings.Contains(s, "directory")
}
