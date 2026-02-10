package detect

import (
	"os"
	"testing"
)

// TestBasicDetectorEmptyBinaries tests detector with no binaries to check
func TestBasicDetectorEmptyBinaries(t *testing.T) {
	detector := &BasicDetector{
		Binaries: []string{},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}
	if len(result.MissingBinaries) != 0 {
		t.Errorf("expected 0 missing binaries, got %d", len(result.MissingBinaries))
	}
}

// TestBasicDetectorCommonBinary tests detector with common binary (e.g., ls)
func TestBasicDetectorCommonBinary(t *testing.T) {
	detector := &BasicDetector{
		Binaries: []string{"ls"},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}
	if len(result.MissingBinaries) != 0 {
		t.Errorf("expected ls binary to be found, got missing: %v", result.MissingBinaries)
	}
}

// TestBasicDetectorMissingBinary tests detector with non-existent binary
func TestBasicDetectorMissingBinary(t *testing.T) {
	detector := &BasicDetector{
		Binaries: []string{"nonexistent_binary_xyz_123"},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}
	if len(result.MissingBinaries) != 1 {
		t.Errorf("expected 1 missing binary, got %d", len(result.MissingBinaries))
	}
	if result.MissingBinaries[0] != "nonexistent_binary_xyz_123" {
		t.Errorf("expected nonexistent_binary_xyz_123 in missing, got %v", result.MissingBinaries)
	}
}

// TestBasicDetectorMultipleBinaries tests detector with multiple binaries
func TestBasicDetectorMultipleBinaries(t *testing.T) {
	detector := &BasicDetector{
		Binaries: []string{"ls", "cat", "grep", "nonexistent_xyz"},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}

	// Should find ls, cat, grep but not nonexistent_xyz
	if len(result.MissingBinaries) != 1 {
		t.Errorf("expected 1 missing binary, got %d: %v", len(result.MissingBinaries), result.MissingBinaries)
	}
}

// TestBasicDetectorWithPATH tests detector respects PATH environment variable
func TestBasicDetectorWithPATH(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)

	// Set PATH to empty
	os.Setenv("PATH", "")

	detector := &BasicDetector{
		Binaries: []string{"ls"},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}

	// Should have warning about empty PATH and not find ls
	hasPathWarning := false
	for _, w := range result.Warnings {
		if w == "PATH is empty" {
			hasPathWarning = true
			break
		}
	}
	if !hasPathWarning {
		t.Errorf("expected PATH is empty warning, got warnings: %v", result.Warnings)
	}

	if len(result.MissingBinaries) != 1 {
		t.Errorf("expected ls to be missing with empty PATH, got missing: %v", result.MissingBinaries)
	}
}

// TestBasicDetectorNormalPATH tests detector with normal PATH
func TestBasicDetectorNormalPATH(t *testing.T) {
	path := os.Getenv("PATH")
	if path == "" {
		t.Skip("PATH environment variable is not set")
	}

	detector := &BasicDetector{
		Binaries: []string{"ls"},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}

	// Should not have PATH warning with normal PATH
	for _, w := range result.Warnings {
		if w == "PATH is empty" || w == "PATH has no entries" {
			t.Errorf("unexpected warning with normal PATH: %s", w)
		}
	}
}

// TestBasicDetectorResultStructure tests that Result struct is properly initialized
func TestBasicDetectorResultStructure(t *testing.T) {
	detector := &BasicDetector{
		Binaries: []string{},
	}

	result, err := detector.Detect()
	if err != nil {
		t.Errorf("expected detection to succeed, got error: %v", err)
	}

	if result.MissingBinaries == nil {
		t.Errorf("expected MissingBinaries slice to be initialized, got nil")
	}
	if result.BrokenBinaries == nil {
		t.Errorf("expected BrokenBinaries slice to be initialized, got nil")
	}
	if result.Warnings == nil {
		t.Errorf("expected Warnings slice to be initialized, got nil")
	}
}
