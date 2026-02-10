package selfmodel

import (
	"testing"
)

// TestComputeLawMetricsEmpty tests metrics calculation on empty string
func TestComputeLawMetricsEmpty(t *testing.T) {
	metrics := ComputeLawMetrics("")

	if metrics.RuleLines != 0 {
		t.Errorf("expected 0 rule lines for empty string, got %d", metrics.RuleLines)
	}
	if metrics.ConstraintCount != 0 {
		t.Errorf("expected 0 constraints for empty string, got %d", metrics.ConstraintCount)
	}
	if metrics.EnforcementActive {
		t.Errorf("expected enforcement to be false for empty string")
	}
}

// TestComputeLawMetricsBlankLines tests that blank lines are ignored
func TestComputeLawMetricsBlankLines(t *testing.T) {
	raw := "\n\n\n"
	metrics := ComputeLawMetrics(raw)

	if metrics.RuleLines != 0 {
		t.Errorf("expected 0 rule lines for blank lines, got %d", metrics.RuleLines)
	}
}

// TestComputeLawMetricsSimpleRule tests metrics with simple rule
func TestComputeLawMetricsSimpleRule(t *testing.T) {
	raw := "rule 1\nrule 2"
	metrics := ComputeLawMetrics(raw)

	if metrics.RuleLines != 2 {
		t.Errorf("expected 2 rule lines, got %d", metrics.RuleLines)
	}
}

// TestComputeLawMetricsConstraintDetection tests constraint keyword detection
func TestComputeLawMetricsConstraintDetection(t *testing.T) {
	testCases := []struct {
		name      string
		keyword   string
		expectMax int
	}{
		{"must not", "must not", 1},
		{"must", "must", 1},
		{"never", "never", 1},
		{"refuse", "refuse", 1},
		{"forbidden", "forbidden", 1},
		{"non-negotiable", "non-negotiable", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			raw := "rule: " + tc.keyword
			metrics := ComputeLawMetrics(raw)

			if metrics.ConstraintCount < 1 {
				t.Errorf("expected at least 1 constraint for keyword '%s', got %d", tc.keyword, metrics.ConstraintCount)
			}
		})
	}
}

// TestComputeLawMetricsMultipleConstraints tests multiple constraints in one line
func TestComputeLawMetricsMultipleConstraints(t *testing.T) {
	raw := "rule: must never refuse"
	metrics := ComputeLawMetrics(raw)

	if metrics.ConstraintCount < 2 {
		t.Errorf("expected at least 2 constraints in 'must never refuse', got %d", metrics.ConstraintCount)
	}
}

// TestComputeLawMetricsEnforcementDetection tests enforcement keyword detection
func TestComputeLawMetricsEnforcementDetection(t *testing.T) {
	raw := "May reason about state, but it may never originate state"
	metrics := ComputeLawMetrics(raw)

	if !metrics.EnforcementActive {
		t.Errorf("expected enforcement to be active for enforcement phrase")
	}
}

// TestComputeLawMetricsHumanGreetingIgnored tests that human_greeting lines are ignored
func TestComputeLawMetricsHumanGreetingIgnored(t *testing.T) {
	raw := "human_greeting: Hello there\nrule: must"
	metrics := ComputeLawMetrics(raw)

	if metrics.RuleLines != 1 {
		t.Errorf("expected 1 rule line (human greeting ignored), got %d", metrics.RuleLines)
	}
}

// TestComputeLawMetricsMultilineModel tests metrics on realistic model
func TestComputeLawMetricsMultilineModel(t *testing.T) {
	raw := `human_greeting: Welcome
rule1: must not violate constraints
rule2: forbidden behaviour
constraint: never
constraint2: refuse`
	metrics := ComputeLawMetrics(raw)

	if metrics.RuleLines <= 0 {
		t.Errorf("expected positive rule lines for realistic model")
	}
	if metrics.ConstraintCount <= 0 {
		t.Errorf("expected positive constraint count for realistic model")
	}
}

// TestComputeLawMetricsStructureInitialization tests that LawMetrics struct is initialized
func TestComputeLawMetricsStructureInitialization(t *testing.T) {
	raw := "any content"
	metrics := ComputeLawMetrics(raw)

	if metrics.RuleLines < 0 {
		t.Errorf("expected non-negative RuleLines")
	}
	if metrics.ConstraintCount < 0 {
		t.Errorf("expected non-negative ConstraintCount")
	}
}

// TestComputeLawMetricsCaseSensitivity tests that keyword matching is case-insensitive
func TestComputeLawMetricsCaseSensitivity(t *testing.T) {
	raw1 := "rule: MUST NOT"
	raw2 := "rule: must not"

	metrics1 := ComputeLawMetrics(raw1)
	metrics2 := ComputeLawMetrics(raw2)

	if metrics1.ConstraintCount != metrics2.ConstraintCount {
		t.Errorf("expected case-insensitive matching: %d vs %d", metrics1.ConstraintCount, metrics2.ConstraintCount)
	}
}

// TestComputeLawMetricsWhitespace tests that whitespace is handled correctly
func TestComputeLawMetricsWhitespace(t *testing.T) {
	raw := "  rule with spaces  \n\t\trule with tabs\t\t"
	metrics := ComputeLawMetrics(raw)

	if metrics.RuleLines != 2 {
		t.Errorf("expected 2 rule lines with whitespace handling, got %d", metrics.RuleLines)
	}
}
