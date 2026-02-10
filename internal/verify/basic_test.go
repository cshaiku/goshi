package verify

import (
	"testing"
)

// TestBasicVerifierNoBinaries tests verifier with no binaries to check
func TestBasicVerifierNoBinaries(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to succeed, got error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected verification to pass with no binaries")
	}
	if len(result.Failures) != 0 {
		t.Errorf("expected 0 failures, got %d", len(result.Failures))
	}
}

// TestBasicVerifierCommonBinary tests verifier with common binary (should pass)
func TestBasicVerifierCommonBinary(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{"ls"},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to succeed, got error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected verification to pass for common binary 'ls'")
	}
	if len(result.Failures) != 0 {
		t.Errorf("expected 0 failures for common binary, got: %v", result.Failures)
	}
}

// TestBasicVerifierMissingBinary tests verifier with missing binary (should fail)
func TestBasicVerifierMissingBinary(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{"nonexistent_binary_xyz_123"},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to return error info, got error: %v", err)
	}
	if result.Passed {
		t.Errorf("expected verification to fail for missing binary")
	}
	if len(result.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(result.Failures))
	}
}

// TestBasicVerifierMultipleBinaries tests verifier with multiple binaries
func TestBasicVerifierMultipleBinaries(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{"ls", "cat", "grep", "nonexistent_xyz_123"},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to succeed, got error: %v", err)
	}
	if result.Passed {
		t.Errorf("expected verification to fail with missing binary")
	}
	if len(result.Failures) != 1 {
		t.Errorf("expected 1 failure for missing binary, got %d: %v", len(result.Failures), result.Failures)
	}
}

// TestBasicVerifierAllCommonBinaries tests verifier with all common binaries
func TestBasicVerifierAllCommonBinaries(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{"ls", "cat", "grep", "echo"},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to succeed, got error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected verification to pass for common binaries, got failures: %v", result.Failures)
	}
	if len(result.Failures) != 0 {
		t.Errorf("expected 0 failures for common binaries, got %d", len(result.Failures))
	}
}

// TestBasicVerifierResultStructure tests that Result struct is properly initialized
func TestBasicVerifierResultStructure(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to succeed, got error: %v", err)
	}

	if result.Failures == nil {
		t.Errorf("expected Failures slice to be initialized, got nil")
	}
}

// TestBasicVerifierFailureMessage tests that failure messages are properly formatted
func TestBasicVerifierFailureMessage(t *testing.T) {
	verifier := &BasicVerifier{
		Binaries: []string{"missing_bin_xyz"},
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Errorf("expected verification to succeed, got error: %v", err)
	}

	if len(result.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(result.Failures))
	}

	failure := result.Failures[0]
	if !contains(failure, "missing_bin_xyz") {
		t.Errorf("expected failure message to contain 'missing_bin_xyz', got: %s", failure)
	}
}

// Helper function
func contains(s, substring string) bool {
	for i := 0; i <= len(s)-len(substring); i++ {
		if s[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
