package detect

type Capability string

const (
	CapabilityFSRead  Capability = "fs_read"
	CapabilityFSWrite Capability = "fs_write"
)

type Rule struct {
	Capability Capability
	Verbs      []string
	Nouns      []string
	Window     int
}

// DetectCapabilities evaluates rules against tokenized input.
func DetectCapabilities(prompt string, rules []Rule) []Capability {
	tokens := Tokenize(prompt)

	var detected []Capability

	for _, rule := range rules {
		if matchRule(tokens, rule) {
			detected = append(detected, rule.Capability)
		}
	}

	return detected
}

func matchRule(tokens []string, rule Rule) bool {
	for i, tok := range tokens {
		if !contains(rule.Verbs, tok) {
			continue
		}

		start := i - rule.Window
		end := i + rule.Window

		if start < 0 {
			start = 0
		}
		if end >= len(tokens) {
			end = len(tokens) - 1
		}

		for j := start; j <= end; j++ {
			if contains(rule.Nouns, tokens[j]) {
				return true
			}
		}
	}

	return false
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
