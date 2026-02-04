package fs

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	ErrApplyFailed = errors.New("failed to apply write proposal")
)

// ApplyWrite applies a WriteProposal to disk safely.
// It prefers git-based application when inside a git repo.
// This function MUTATES the filesystem.
func ApplyWrite(g *Guard, p *WriteProposal) error {
	if p == nil {
		return ErrApplyFailed
	}

	// Ensure path is still valid and within root
	resolved, err := g.Resolve(p.Path)
	if err != nil {
		return err
	}

	// Try git apply first if possible
	if canGitApply(resolved) {
		if err := gitApply(p.Diff, filepath.Dir(resolved)); err == nil {
			return nil
		}
		// fall through to atomic write if git apply fails
	}

	// Atomic write fallback
	dir := filepath.Dir(resolved)
	tmp, err := os.CreateTemp(dir, ".goshi-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.WriteString(p.Proposed); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	// Preserve permissions if file existed
	if !p.IsNewFile {
		if info, err := os.Stat(resolved); err == nil {
			_ = os.Chmod(tmp.Name(), info.Mode())
		}
	}

	return os.Rename(tmp.Name(), resolved)
}

func canGitApply(path string) bool {
	dir := filepath.Dir(path)
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}

func gitApply(diff string, dir string) error {
	cmd := exec.Command("git", "-C", dir, "apply", "--whitespace=nowarn", "-")
	cmd.Stdin = bytes.NewBufferString(diff)
	return cmd.Run()
}
