package integrity

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TestHelper provides utilities for offensive security testing
type TestHelper struct {
	RepoRoot     string
	ManifestPath string
}

// NewTestHelper creates a test helper for the current repository
func NewTestHelper() (*TestHelper, error) {
	repoRoot := findRepoRoot()
	if repoRoot == "." {
		return nil, fmt.Errorf("not in git repository")
	}

	return &TestHelper{
		RepoRoot:     repoRoot,
		ManifestPath: filepath.Join(repoRoot, "goshi.sum"),
	}, nil
}

// RandomGoFile selects a random .go file from the manifest
func (h *TestHelper) RandomGoFile() (string, error) {
	diag := &IntegrityDiagnostic{
		ManifestPath: h.ManifestPath,
		RepoRoot:     h.RepoRoot,
	}

	entries, err := diag.parseManifest()
	if err != nil {
		return "", fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Filter for .go files only
	var goFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.FilePath, ".go") {
			goFiles = append(goFiles, entry.FilePath)
		}
	}

	if len(goFiles) == 0 {
		return "", fmt.Errorf("no .go files found in manifest")
	}

	// Select random file
	rand.Seed(time.Now().UnixNano())
	return goFiles[rand.Intn(len(goFiles))], nil
}

// BackupFile creates a temporary backup of a file
func (h *TestHelper) BackupFile(relPath string) (tempPath string, err error) {
	srcPath := filepath.Join(h.RepoRoot, relPath)

	// Read file content
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "goshi-backup-*.go")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpPath, nil
}

// RestoreFile restores a file from a backup
func (h *TestHelper) RestoreFile(relPath, backupPath string) error {
	destPath := filepath.Join(h.RepoRoot, relPath)

	// Read backup content
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	// Write back to original location
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	// Clean up backup
	os.Remove(backupPath)

	return nil
}

// TamperWithFile backs up a file, modifies it, and returns a restore function
func (h *TestHelper) TamperWithFile(relPath string) (restore func() error, err error) {
	// Create backup
	backupPath, err := h.BackupFile(relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to backup file: %w", err)
	}

	// Modify the file
	srcPath := filepath.Join(h.RepoRoot, relPath)
	content, err := os.ReadFile(srcPath)
	if err != nil {
		os.Remove(backupPath)
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Add an innocuous comment to change the hash
	tampered := append([]byte("// TAMPERED FOR TESTING\n"), content...)
	if err := os.WriteFile(srcPath, tampered, 0644); err != nil {
		os.Remove(backupPath)
		return nil, fmt.Errorf("failed to tamper with file: %w", err)
	}

	// Return restore function
	restore = func() error {
		return h.RestoreFile(relPath, backupPath)
	}

	return restore, nil
}

// DeleteFile temporarily deletes a file and returns a restore function
func (h *TestHelper) DeleteFile(relPath string) (restore func() error, err error) {
	// Create backup
	backupPath, err := h.BackupFile(relPath)
	if err != nil {
		return nil, fmt.Errorf("failed to backup file: %w", err)
	}

	// Delete the file
	srcPath := filepath.Join(h.RepoRoot, relPath)
	if err := os.Remove(srcPath); err != nil {
		os.Remove(backupPath)
		return nil, fmt.Errorf("failed to delete file: %w", err)
	}

	// Return restore function
	restore = func() error {
		return h.RestoreFile(relPath, backupPath)
	}

	return restore, nil
}

// ModifyFile adds a comment to a file to change its hash
func (h *TestHelper) ModifyFile(relPath string, modification string) error {
	srcPath := filepath.Join(h.RepoRoot, relPath)

	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Add modification at the beginning
	modified := append([]byte(modification+"\n"), content...)
	if err := os.WriteFile(srcPath, modified, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// CreateFakeFile creates a fake .go file not in the manifest
func (h *TestHelper) CreateFakeFile(relPath string) (cleanup func() error, err error) {
	fakeContent := `package fake

// This is a fake file created for testing
// It should not exist in the manifest

func FakeFunction() {
	// Do nothing
}
`

	fakeFilePath := filepath.Join(h.RepoRoot, relPath)

	// Ensure directory exists
	dir := filepath.Dir(fakeFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Create fake file
	if err := os.WriteFile(fakeFilePath, []byte(fakeContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create fake file: %w", err)
	}

	// Return cleanup function
	cleanup = func() error {
		return os.Remove(fakeFilePath)
	}

	return cleanup, nil
}
