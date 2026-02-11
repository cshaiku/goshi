package integrity

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cshaiku/goshi/internal/diagnose"
)

// IntegrityDiagnostic checks the integrity of source files against a manifest
type IntegrityDiagnostic struct {
	ManifestPath string
	RepoRoot     string
}

// ManifestEntry represents a single entry in the integrity manifest
type ManifestEntry struct {
	Algorithm string
	Hash      string
	FilePath  string
}

// VerificationResult contains the results of file verification
type VerificationResult struct {
	TotalFiles    int
	VerifiedFiles int
	MissingFiles  []string
	ModifiedFiles []FileModification
}

// FileModification represents a file that has been modified
type FileModification struct {
	Path         string
	ExpectedHash string
	ActualHash   string
}

// NewIntegrityDiagnostic creates a new integrity diagnostic with default paths
func NewIntegrityDiagnostic() *IntegrityDiagnostic {
	repoRoot := findRepoRoot()
	manifestPath := filepath.Join(repoRoot, "goshi.sum")

	return &IntegrityDiagnostic{
		ManifestPath: manifestPath,
		RepoRoot:     repoRoot,
	}
}

// Run executes the integrity diagnostic and returns issues
func (d *IntegrityDiagnostic) Run() []diagnose.Issue {
	var issues []diagnose.Issue

	// Check if manifest exists
	if _, err := os.Stat(d.ManifestPath); os.IsNotExist(err) {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_NO_MANIFEST",
			Message:  fmt.Sprintf("No integrity manifest found at %s", d.ManifestPath),
			Strategy: "Run 'scripts/generate_goshi_sum.sh' to create the manifest",
			Severity: diagnose.SeverityWarn,
		})
		return issues
	}

	// Parse manifest
	entries, err := d.parseManifest()
	if err != nil {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_PARSE_ERROR",
			Message:  fmt.Sprintf("Failed to parse integrity manifest: %v", err),
			Severity: diagnose.SeverityError,
		})
		return issues
	}

	// Verify files
	result := d.verifyFiles(entries)

	// Report missing files
	if len(result.MissingFiles) > 0 {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_MISSING_FILES",
			Message:  fmt.Sprintf("%d tracked files are missing:\n%s", len(result.MissingFiles), strings.Join(result.MissingFiles, "\n")),
			Strategy: "Files may have been deleted or moved. Regenerate goshi.sum if this is intentional.",
			Severity: diagnose.SeverityError,
		})
	}

	// Report modified files
	if len(result.ModifiedFiles) > 0 {
		modifiedList := make([]string, len(result.ModifiedFiles))
		for i, mod := range result.ModifiedFiles {
			modifiedList[i] = fmt.Sprintf("  %s\n    Expected: %s\n    Actual:   %s",
				mod.Path, mod.ExpectedHash[:16]+"...", mod.ActualHash[:16]+"...")
		}
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_HASH_MISMATCH",
			Message:  fmt.Sprintf("%d files have been modified:\n%s", len(result.ModifiedFiles), strings.Join(modifiedList, "\n")),
			Strategy: "Review changes and regenerate goshi.sum after committing valid changes.",
			Severity: diagnose.SeverityError,
		})
	}

	// If all files verified, add success issue
	if len(issues) == 0 {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_OK",
			Message:  fmt.Sprintf("All %d files verified successfully. Source file integrity check passed.", result.VerifiedFiles),
			Severity: diagnose.SeverityOK,
		})
	}

	return issues
}

// parseManifest reads and parses the integrity manifest file
func (d *IntegrityDiagnostic) parseManifest() ([]ManifestEntry, error) {
	file, err := os.Open(d.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer file.Close()

	var entries []ManifestEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line: ALGORITHM HASH FILEPATH
		parts := strings.SplitN(line, " ", 3)
		if len(parts) != 3 {
			continue // Skip malformed lines
		}

		entries = append(entries, ManifestEntry{
			Algorithm: parts[0],
			Hash:      parts[1],
			FilePath:  parts[2],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading manifest: %w", err)
	}

	return entries, nil
}

// verifyFiles checks each file in the entries against its expected hash
func (d *IntegrityDiagnostic) verifyFiles(entries []ManifestEntry) VerificationResult {
	result := VerificationResult{
		TotalFiles:    len(entries),
		MissingFiles:  []string{},
		ModifiedFiles: []FileModification{},
	}

	for _, entry := range entries {
		fullPath := filepath.Join(d.RepoRoot, entry.FilePath)

		// Check if file exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			result.MissingFiles = append(result.MissingFiles, entry.FilePath)
			continue
		}

		// Compute actual hash
		actualHash, err := computeSHA256(fullPath)
		if err != nil {
			// Treat read errors as missing
			result.MissingFiles = append(result.MissingFiles, entry.FilePath)
			continue
		}

		// Compare hashes
		if actualHash != entry.Hash {
			result.ModifiedFiles = append(result.ModifiedFiles, FileModification{
				Path:         entry.FilePath,
				ExpectedHash: entry.Hash,
				ActualHash:   actualHash,
			})
			continue
		}

		result.VerifiedFiles++
	}

	return result
}

// computeSHA256 calculates the SHA256 hash of a file
func computeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// findRepoRoot walks up the directory tree to find the repository root
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "."
		}
		dir = parent
	}
}
