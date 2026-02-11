package integrity

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PlanRepair loads the manifest, validates the tarball, and returns verification results.
func (d *IntegrityDiagnostic) PlanRepair() (Manifest, VerificationResult, error) {
	manifest, err := d.parseManifest()
	if err != nil {
		return Manifest{}, VerificationResult{}, err
	}

	if manifest.Tarball.Path == "" {
		return Manifest{}, VerificationResult{}, errors.New("manifest missing tarball metadata")
	}

	tarballPath := filepath.Join(d.RepoRoot, manifest.Tarball.Path)
	if _, err := os.Stat(tarballPath); err != nil {
		return Manifest{}, VerificationResult{}, fmt.Errorf("tarball missing: %w", err)
	}

	tarballHash, err := computeSHA256(tarballPath)
	if err != nil {
		return Manifest{}, VerificationResult{}, fmt.Errorf("tarball unreadable: %w", err)
	}

	if tarballHash != manifest.Tarball.Hash {
		return Manifest{}, VerificationResult{}, errors.New("tarball hash mismatch")
	}

	result := d.verifyFiles(manifest.Files)
	return manifest, result, nil
}

// RestoreFromTarball extracts the requested files from the source tarball.
func (d *IntegrityDiagnostic) RestoreFromTarball(manifest Manifest, targets []string) ([]string, error) {
	if len(targets) == 0 {
		return nil, nil
	}

	targetSet := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		clean := filepath.Clean(target)
		targetSet[clean] = struct{}{}
	}

	tarballPath := filepath.Join(d.RepoRoot, manifest.Tarball.Path)
	archive, err := os.Open(tarballPath)
	if err != nil {
		return nil, fmt.Errorf("open tarball: %w", err)
	}
	defer archive.Close()

	gz, err := gzip.NewReader(archive)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	var restored []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return restored, fmt.Errorf("read tar: %w", err)
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		name := filepath.Clean(hdr.Name)
		if filepath.IsAbs(name) || strings.HasPrefix(name, "..") {
			return restored, fmt.Errorf("unsafe tar entry: %s", hdr.Name)
		}

		if _, ok := targetSet[name]; !ok {
			continue
		}

		destPath := filepath.Join(d.RepoRoot, name)
		if !strings.HasPrefix(destPath, d.RepoRoot+string(os.PathSeparator)) && destPath != d.RepoRoot {
			return restored, fmt.Errorf("path traversal detected: %s", name)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return restored, fmt.Errorf("mkdir: %w", err)
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, hdr.FileInfo().Mode().Perm())
		if err != nil {
			return restored, fmt.Errorf("write file: %w", err)
		}

		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return restored, fmt.Errorf("copy file: %w", err)
		}

		if err := out.Close(); err != nil {
			return restored, fmt.Errorf("close file: %w", err)
		}

		restored = append(restored, name)
	}

	return restored, nil
}
