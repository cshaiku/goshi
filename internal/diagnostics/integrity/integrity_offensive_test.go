//go:build offensive

package integrity

import (
	"path/filepath"
	"testing"

	"github.com/cshaiku/goshi/internal/diagnose"
)

// TestIntegrityDetectsTampering verifies that file modification is detected
func TestIntegrityDetectsTampering(t *testing.T) {
	helper, err := NewTestHelper()
	if err != nil {
		t.Skip("Not in git repository, skipping offensive test")
	}

	// Select a random .go file
	targetFile, err := helper.RandomGoFile()
	if err != nil {
		t.Fatalf("Failed to select random file: %v", err)
	}

	t.Logf("Testing with file: %s", targetFile)

	// Tamper with the file
	restore, err := helper.TamperWithFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to tamper with file: %v", err)
	}

	// Ensure restoration happens even if test fails
	defer func() {
		if err := restore(); err != nil {
			t.Errorf("Failed to restore file: %v", err)
		}
	}()

	// Run integrity check
	diag := NewIntegrityDiagnostic()
	issues := diag.Run()

	// Should detect modification
	foundHashMismatch := false
	for _, issue := range issues {
		if issue.Code == "INTEGRITY_HASH_MISMATCH" {
			foundHashMismatch = true
			t.Logf("✓ Detected tampering: %s", issue.Message)
			break
		}
	}

	if !foundHashMismatch {
		t.Errorf("Failed to detect file tampering - SECURITY ISSUE!")
		t.Logf("Issues returned: %+v", issues)
	}
}

// TestIntegrityDetectsMissingFile verifies that deleted files are detected
func TestIntegrityDetectsMissingFile(t *testing.T) {
	helper, err := NewTestHelper()
	if err != nil {
		t.Skip("Not in git repository, skipping offensive test")
	}

	// Select a random .go file
	targetFile, err := helper.RandomGoFile()
	if err != nil {
		t.Fatalf("Failed to select random file: %v", err)
	}

	t.Logf("Testing with file: %s", targetFile)

	// Delete the file
	restore, err := helper.DeleteFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}

	// Ensure restoration happens even if test fails
	defer func() {
		if err := restore(); err != nil {
			t.Errorf("Failed to restore file: %v", err)
		}
	}()

	// Run integrity check
	diag := NewIntegrityDiagnostic()
	issues := diag.Run()

	// Should detect missing file
	foundMissing := false
	for _, issue := range issues {
		if issue.Code == "INTEGRITY_MISSING_FILES" {
			foundMissing = true
			t.Logf("✓ Detected missing file: %s", issue.Message)
			break
		}
	}

	if !foundMissing {
		t.Errorf("Failed to detect missing file - SECURITY ISSUE!")
		t.Logf("Issues returned: %+v", issues)
	}
}

// TestIntegrityRestoresMissingFileFromTarball verifies missing file recovery from the tarball.
func TestIntegrityRestoresMissingFileFromTarball(t *testing.T) {
	helper, err := NewTestHelper()
	if err != nil {
		t.Skip("Not in git repository, skipping offensive test")
	}

	// Select a random .go file
	targetFile, err := helper.RandomGoFile()
	if err != nil {
		t.Fatalf("Failed to select random file: %v", err)
	}

	// Delete the file
	restore, err := helper.DeleteFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to delete file: %v", err)
	}
	defer func() {
		if err := restore(); err != nil {
			t.Errorf("Failed to restore file: %v", err)
		}
	}()

	diag := NewIntegrityDiagnostic()
	manifest, result, err := diag.PlanRepair()
	if err != nil {
		t.Skipf("Integrity reference bundle unavailable: %v", err)
	}

	missing := false
	for _, path := range result.MissingFiles {
		if path == targetFile {
			missing = true
			break
		}
	}
	if !missing {
		t.Fatalf("Expected missing file %s to be detected", targetFile)
	}

	restored, err := diag.RestoreFromTarball(manifest, []string{targetFile})
	if err != nil {
		t.Fatalf("Failed to restore from tarball: %v", err)
	}
	if len(restored) != 1 || restored[0] != targetFile {
		t.Fatalf("Unexpected restore result: %v", restored)
	}

	var expectedHash string
	for _, entry := range manifest.Files {
		if entry.FilePath == targetFile {
			expectedHash = entry.Hash
			break
		}
	}
	if expectedHash == "" {
		t.Fatalf("Missing manifest entry for %s", targetFile)
	}

	actualHash, err := computeSHA256(filepath.Join(diag.RepoRoot, targetFile))
	if err != nil {
		t.Fatalf("Failed to hash restored file: %v", err)
	}
	if actualHash != expectedHash {
		t.Fatalf("Restored file hash mismatch: expected %s got %s", expectedHash, actualHash)
	}
}

// TestIntegrityPassesWhenClean verifies no false positives on clean repo
func TestIntegrityPassesWhenClean(t *testing.T) {
	helper, err := NewTestHelper()
	if err != nil {
		t.Skip("Not in git repository, skipping offensive test")
	}

	t.Logf("Testing clean repository at: %s", helper.RepoRoot)

	// Run integrity check on clean repo
	diag := NewIntegrityDiagnostic()
	issues := diag.Run()

	// Should have no integrity issues
	hasIntegrityIssues := false
	for _, issue := range issues {
		if issue.Code == "INTEGRITY_HASH_MISMATCH" ||
			issue.Code == "INTEGRITY_MISSING_FILES" {
			hasIntegrityIssues = true
			t.Errorf("False positive detected: %s - %s", issue.Code, issue.Message)
		}
	}

	if hasIntegrityIssues {
		t.Errorf("Integrity check reported false positives on clean repository")
	} else {
		t.Log("✓ Clean repository passed integrity check")
	}
}

// TestMultipleModifications verifies detection of multiple tampered files
func TestMultipleModifications(t *testing.T) {
	helper, err := NewTestHelper()
	if err != nil {
		t.Skip("Not in git repository, skipping offensive test")
	}

	// Get multiple files
	file1, err := helper.RandomGoFile()
	if err != nil {
		t.Fatalf("Failed to select first file: %v", err)
	}

	file2, err := helper.RandomGoFile()
	if err != nil {
		t.Fatalf("Failed to select second file: %v", err)
	}

	// Ensure different files
	if file1 == file2 {
		t.Skip("Got same file twice, skipping")
	}

	t.Logf("Testing with files: %s, %s", file1, file2)

	// Tamper with first file
	restore1, err := helper.TamperWithFile(file1)
	if err != nil {
		t.Fatalf("Failed to tamper with file 1: %v", err)
	}
	defer restore1()

	// Tamper with second file
	restore2, err := helper.TamperWithFile(file2)
	if err != nil {
		t.Fatalf("Failed to tamper with file 2: %v", err)
	}
	defer restore2()

	// Run integrity check
	diag := NewIntegrityDiagnostic()
	issues := diag.Run()

	// Should detect modifications
	foundHashMismatch := false
	for _, issue := range issues {
		if issue.Code == "INTEGRITY_HASH_MISMATCH" {
			foundHashMismatch = true
			t.Logf("✓ Detected multiple modifications: %s", issue.Message)
			// Check that it reports at least 2 files
			if !contains(issue.Message, "2 files") {
				t.Logf("Warning: Expected '2 files' in message, got: %s", issue.Message)
			}
			break
		}
	}

	if !foundHashMismatch {
		t.Errorf("Failed to detect multiple file modifications")
	}
}

// TestSeverityLevels verifies that integrity issues have appropriate severity
func TestSeverityLevels(t *testing.T) {
	helper, err := NewTestHelper()
	if err != nil {
		t.Skip("Not in git repository, skipping offensive test")
	}

	targetFile, err := helper.RandomGoFile()
	if err != nil {
		t.Fatalf("Failed to select random file: %v", err)
	}

	restore, err := helper.TamperWithFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to tamper with file: %v", err)
	}
	defer restore()

	// Run integrity check
	diag := NewIntegrityDiagnostic()
	issues := diag.Run()

	// Verify severity is ERROR (not WARN, not FATAL)
	for _, issue := range issues {
		if issue.Code == "INTEGRITY_HASH_MISMATCH" {
			if issue.Severity != diagnose.SeverityError {
				t.Errorf("Expected severity ERROR, got %v", issue.Severity)
			} else {
				t.Logf("✓ Correct severity: ERROR")
			}
			break
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
