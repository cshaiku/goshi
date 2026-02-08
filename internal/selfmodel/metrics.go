package selfmodel

import (
	"strings"
)

type LawMetrics struct {
	RuleLines        int
	ConstraintCount  int
	EnforcementActive bool
}

// ComputeLawMetrics derives deterministic metrics from the raw self-model text.
func ComputeLawMetrics(raw string) LawMetrics {
	lines := strings.Split(raw, "\n")

	var ruleLines int
	var constraints int
	var enforcement bool

	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}

		// Ignore human greeting content
		if strings.HasPrefix(l, "human_greeting:") {
			continue
		}

		ruleLines++

		ll := strings.ToLower(l)
		for _, kw := range []string{
			"must not",
			"must",
			"never",
			"refuse",
			"forbidden",
			"non-negotiable",
		} {
			if strings.Contains(ll, kw) {
				constraints++
			}
		}

		if strings.Contains(
			ll,
			"may reason about state, but it may never originate state",
		) {
			enforcement = true
		}
	}

	return LawMetrics{
		RuleLines:        ruleLines,
		ConstraintCount:  constraints,
		EnforcementActive: enforcement,
	}
}
