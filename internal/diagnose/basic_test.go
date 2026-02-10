package diagnose

import (
	"testing"

	"github.com/cshaiku/goshi/internal/detect"
)

// TestBasicDiagnoserEmptyDetectResult tests diagnoser with empty detect result
func TestBasicDiagnoserEmptyDetectResult(t *testing.T) {
	diagnoser := &BasicDiagnoser{}
	detectResult := detect.Result{
		MissingBinaries: []string{},
		BrokenBinaries:  []string{},
		Warnings:        []string{},
	}

	result, err := diagnoser.Diagnose(detectResult)
	if err != nil {
		t.Errorf("expected diagnosis to succeed, got error: %v", err)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(result.Issues))
	}
}

// TestBasicDiagnoserWithMissingBinaries tests diagnoser identifies missing binaries
func TestBasicDiagnoserWithMissingBinaries(t *testing.T) {
	diagnoser := &BasicDiagnoser{}
	detectResult := detect.Result{
		MissingBinaries: []string{"git", "python"},
		BrokenBinaries:  []string{},
		Warnings:        []string{},
	}

	result, err := diagnoser.Diagnose(detectResult)
	if err != nil {
		t.Errorf("expected diagnosis to succeed, got error: %v", err)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}

	// Check each issue
	expectedBinaries := map[string]bool{"git": false, "python": false}
	for _, issue := range result.Issues {
		if issue.Code != "missing_binary" {
			t.Errorf("expected code 'missing_binary', got '%s'", issue.Code)
		}
		if issue.Severity != SeverityError {
			t.Errorf("expected severity '%s', got '%s'", SeverityError, issue.Severity)
		}
		// Track which binaries we saw
		for bin := range expectedBinaries {
			if contains(issue.Message, bin) {
				expectedBinaries[bin] = true
			}
		}
	}

	// Verify we saw all expected binaries
	for bin, found := range expectedBinaries {
		if !found {
			t.Errorf("expected to see issue for binary '%s'", bin)
		}
	}
}

// TestBasicDiagnoserWithWarnings tests diagnoser processes warnings
func TestBasicDiagnoserWithWarnings(t *testing.T) {
	diagnoser := &BasicDiagnoser{}
	detectResult := detect.Result{
		MissingBinaries: []string{},
		BrokenBinaries:  []string{},
		Warnings:        []string{"PATH is empty", "SHELL not set"},
	}

	result, err := diagnoser.Diagnose(detectResult)
	if err != nil {
		t.Errorf("expected diagnosis to succeed, got error: %v", err)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(result.Issues))
	}

	for _, issue := range result.Issues {
		if issue.Code != "warning" {
			t.Errorf("expected code 'warning', got '%s'", issue.Code)
		}
		if issue.Severity != SeverityWarn {
			t.Errorf("expected severity '%s', got '%s'", SeverityWarn, issue.Severity)
		}
	}
}

// TestBasicDiagnoserWithBothMissingBinariesAndWarnings tests combined scenarios
func TestBasicDiagnoserWithBothMissingBinariesAndWarnings(t *testing.T) {
	diagnoser := &BasicDiagnoser{}
	detectResult := detect.Result{
		MissingBinaries: []string{"git"},
		BrokenBinaries:  []string{},
		Warnings:        []string{"PATH has no entries"},
	}

	result, err := diagnoser.Diagnose(detectResult)
	if err != nil {
		t.Errorf("expected diagnosis to succeed, got error: %v", err)
	}
	if len(result.Issues) != 2 {
		t.Errorf("expected 2 issues (1 binary + 1 warning), got %d", len(result.Issues))
	}

	// Check we have one error and one warning
	errorCount := 0
	warnCount := 0
	for _, issue := range result.Issues {
		if issue.Severity == SeverityError {
			errorCount++
		} else if issue.Severity == SeverityWarn {
			warnCount++
		}
	}

	if errorCount != 1 {
		t.Errorf("expected 1 error severity issue, got %d", errorCount)
	}
	if warnCount != 1 {
		t.Errorf("expected 1 warn severity issue, got %d", warnCount)
	}
}

// TestBasicDiagnoserIssueStructure tests that issues are properly structured
func TestBasicDiagnoserIssueStructure(t *testing.T) {
	diagnoser := &BasicDiagnoser{}
	detectResult := detect.Result{
		MissingBinaries: []string{"missing_bin"},
		BrokenBinaries:  []string{},
		Warnings:        []string{},
	}

	result, err := diagnoser.Diagnose(detectResult)
	if err != nil {
		t.Errorf("expected diagnosis to succeed, got error: %v", err)
	}

	if len(result.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(result.Issues))
	}

	issue := result.Issues[0]
	if issue.Code == "" {
		t.Errorf("expected non-empty Code field")
	}
	if issue.Message == "" {
		t.Errorf("expected non-empty Message field")
	}
	if issue.Strategy == "" {
		t.Errorf("expected non-empty Strategy field")
	}
	if issue.Severity == "" {
		t.Errorf("expected non-empty Severity field")
	}
}

// TestBasicDiagnoserResultStructure tests that Result struct is properly initialized
func TestBasicDiagnoserResultStructure(t *testing.T) {
	diagnoser := &BasicDiagnoser{}
	detectResult := detect.Result{
		MissingBinaries: []string{},
		BrokenBinaries:  []string{},
		Warnings:        []string{},
	}

	result, err := diagnoser.Diagnose(detectResult)
	if err != nil {
		t.Errorf("expected diagnosis to succeed, got error: %v", err)
	}

	if result.Issues == nil {
		t.Errorf("expected Issues slice to be initialized, got nil")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substring string) bool {
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
