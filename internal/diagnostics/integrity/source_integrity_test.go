package integrity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewIntegrityDiagnostic(t *testing.T) {
	diag := NewIntegrityDiagnostic()

	if diag == nil {
		t.Fatal("NewIntegrityDiagnostic returned nil")
	}

	if diag.ManifestPath == "" {
		t.Error("ManifestPath should not be empty")
	}

	if diag.RepoRoot == "" {
		t.Error("RepoRoot should not be empty")
	}
}

func TestParseManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "test.sum")

	content := `# Test manifest
# Generated: 2026-02-10

SHA256 abc123def456 internal/test/file1.go
SHA256 789ghi012jkl internal/test/file2.go
# Comment line
SHA256 mno345pqr678 main.go
`

	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	diag := &IntegrityDiagnostic{
		ManifestPath: manifestPath,
		RepoRoot:     tmpDir,
	}

	entries, err := diag.parseManifest()
	if err != nil {
		t.Fatalf("parseManifest failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("Expected 3 entries, got %d", len(entries))
	}

	if entries[0].Algorithm != "SHA256" {
		t.Errorf("Expected algorithm SHA256, got %s", entries[0].Algorithm)
	}
	if entries[0].Hash != "abc123def456" {
		t.Errorf("Expected hash abc123def456, got %s", entries[0].Hash)
	}
	if entries[0].FilePath != "internal/test/file1.go" {
		t.Errorf("Expected path internal/test/file1.go, got %s", entries[0].FilePath)
	}
}

func TestComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	content := "Hello, World!\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, err := computeSHA256(testFile)
	if err != nil {
		t.Fatalf("computeSHA256 failed: %v", err)
	}

	expected := "c98c24b677eff44860afea6f493bbaec5bb1c4cbb209c6fc2bbb47f66ff2ad31"
	if hash != expected {
		t.Errorf("Hash mismatch:\nExpected: %s\nGot:      %s", expected, hash)
	}
}

func TestVerifyFiles_AllValid(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash, _ := computeSHA256(testFile)

	entries := []ManifestEntry{
		{
			Algorithm: "SHA256",
			Hash:      hash,
			FilePath:  "test.go",
		},
	}

	diag := &IntegrityDiagnostic{
		RepoRoot: tmpDir,
	}

	result := diag.verifyFiles(entries)

	if result.TotalFiles != 1 {
		t.Errorf("Expected 1 total file, got %d", result.TotalFiles)
	}
	if result.VerifiedFiles != 1 {
		t.Errorf("Expected 1 verified file, got %d", result.VerifiedFiles)
	}
	if len(result.MissingFiles) != 0 {
		t.Errorf("Expected 0 missing files, got %d", len(result.MissingFiles))
	}
	if len(result.ModifiedFiles) != 0 {
		t.Errorf("Expected 0 modified files, got %d", len(result.ModifiedFiles))
	}
}

func TestVerifyFiles_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	entries := []ManifestEntry{
		{
			Algorithm: "SHA256",
			Hash:      "abc123",
			FilePath:  "nonexistent.go",
		},
	}

	diag := &IntegrityDiagnostic{
		RepoRoot: tmpDir,
	}

	result := diag.verifyFiles(entries)

	if len(result.MissingFiles) != 1 {
		t.Errorf("Expected 1 missing file, got %d", len(result.MissingFiles))
	}
	if result.MissingFiles[0] != "nonexistent.go" {
		t.Errorf("Expected missing file nonexistent.go, got %s", result.MissingFiles[0])
	}
}

func TestVerifyFiles_ModifiedFile(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	entries := []ManifestEntry{
		{
			Algorithm: "SHA256",
			Hash:      "wronghash123456",
			FilePath:  "test.go",
		},
	}

	diag := &IntegrityDiagnostic{
		RepoRoot: tmpDir,
	}

	result := diag.verifyFiles(entries)

	if len(result.ModifiedFiles) != 1 {
		t.Errorf("Expected 1 modified file, got %d", len(result.ModifiedFiles))
	}
	if result.ModifiedFiles[0].Path != "test.go" {
		t.Errorf("Expected modified file test.go, got %s", result.ModifiedFiles[0].Path)
	}
	if result.ModifiedFiles[0].ExpectedHash != "wronghash123456" {
		t.Errorf("Expected hash wronghash123456, got %s", result.ModifiedFiles[0].ExpectedHash)
	}
}

func TestRun_NoManifest(t *testing.T) {
	tmpDir := t.TempDir()

	diag := &IntegrityDiagnostic{
		ManifestPath: filepath.Join(tmpDir, "nonexistent.sum"),
		RepoRoot:     tmpDir,
	}

	issues := diag.Run()

	if len(issues) != 1 {
		t.Fatalf("Expected 1 issue, got %d", len(issues))
	}

	if issues[0].Code != "INTEGRITY_NO_MANIFEST" {
		t.Errorf("Expected code INTEGRITY_NO_MANIFEST, got %s", issues[0].Code)
	}
}
