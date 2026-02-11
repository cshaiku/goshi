package integrity

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
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
	Hash     string
	Size     int64
	Mode     string
	ModTime  string
	FilePath string
}

// ManifestTarball represents the tarball metadata stored in the manifest
type ManifestTarball struct {
	Hash string
	Size int64
	Path string
}

// Manifest represents the parsed integrity manifest
type Manifest struct {
	Version       string
	SchemaVersion string
	FormatVersion string
	RootID        string
	Tarball       ManifestTarball
	Files         []ManifestEntry
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
	manifestPath := filepath.Join(repoRoot, ".goshi", "goshi.manifest")

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
			Strategy: "Run 'scripts/generate_goshi_manifest.sh' to create the reference bundle",
			Severity: diagnose.SeverityWarn,
		})
		return issues
	}

	// Parse manifest
	manifest, err := d.parseManifest()
	if err != nil {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_PARSE_ERROR",
			Message:  fmt.Sprintf("Failed to parse integrity manifest: %v", err),
			Severity: diagnose.SeverityError,
		})
		return issues
	}

	// Verify tarball
	if manifest.Tarball.Path == "" {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_TARBALL_NOT_DECLARED",
			Message:  "Integrity manifest is missing tarball metadata",
			Strategy: "Regenerate the reference bundle with 'scripts/generate_goshi_manifest.sh'",
			Severity: diagnose.SeverityError,
		})
		return issues
	}

	tarballPath := filepath.Join(d.RepoRoot, manifest.Tarball.Path)
	if _, err := os.Stat(tarballPath); os.IsNotExist(err) {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_TARBALL_MISSING",
			Message:  fmt.Sprintf("Source tarball missing at %s", tarballPath),
			Strategy: "Regenerate the reference bundle with 'scripts/generate_goshi_manifest.sh'",
			Severity: diagnose.SeverityError,
		})
		return issues
	}

	tarballHash, err := computeSHA256(tarballPath)
	if err != nil {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_TARBALL_UNREADABLE",
			Message:  fmt.Sprintf("Failed to read tarball: %v", err),
			Severity: diagnose.SeverityError,
		})
		return issues
	}

	if tarballHash != manifest.Tarball.Hash {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_TARBALL_HASH_MISMATCH",
			Message:  "Source tarball hash does not match manifest",
			Strategy: "Regenerate the reference bundle with 'scripts/generate_goshi_manifest.sh'",
			Severity: diagnose.SeverityError,
		})
		return issues
	}

	// Verify files
	result := d.verifyFiles(manifest.Files)

	// Report missing files
	if len(result.MissingFiles) > 0 {
		issues = append(issues, diagnose.Issue{
			Code:     "INTEGRITY_MISSING_FILES",
			Message:  fmt.Sprintf("%d tracked files are missing:\n%s", len(result.MissingFiles), strings.Join(result.MissingFiles, "\n")),
			Strategy: "Files may have been deleted or moved. Regenerate the reference bundle if this is intentional.",
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
			Strategy: "Review changes and regenerate the reference bundle after committing valid changes.",
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
func (d *IntegrityDiagnostic) parseManifest() (Manifest, error) {
	file, err := os.Open(d.ManifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer file.Close()

	manifest := Manifest{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "VERSION":
			if len(fields) >= 2 {
				manifest.Version = fields[1]
			}
		case "SCHEMA_VERSION":
			if len(fields) >= 2 {
				manifest.SchemaVersion = fields[1]
			}
		case "FORMAT_VERSION":
			if len(fields) >= 2 {
				manifest.FormatVersion = fields[1]
			}
		case "ROOT_ID":
			if len(fields) >= 2 {
				manifest.RootID = fields[1]
			}
		case "TARBALL":
			if len(fields) >= 4 {
				size, err := strconv.ParseInt(fields[2], 10, 64)
				if err != nil {
					continue
				}
				manifest.Tarball = ManifestTarball{
					Hash: fields[1],
					Size: size,
					Path: strings.Join(fields[3:], " "),
				}
			}
		case "FILE":
			if len(fields) >= 6 {
				size, err := strconv.ParseInt(fields[2], 10, 64)
				if err != nil {
					continue
				}
				manifest.Files = append(manifest.Files, ManifestEntry{
					Hash:     fields[1],
					Size:     size,
					Mode:     fields[3],
					ModTime:  fields[4],
					FilePath: strings.Join(fields[5:], " "),
				})
			}
		default:
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return Manifest{}, fmt.Errorf("error reading manifest: %w", err)
	}

	return manifest, nil
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
