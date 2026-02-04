package fs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNoChangeProposed = errors.New("proposed content is identical to existing content")
)

// WriteProposal represents a non-destructive proposed change to a file.
type WriteProposal struct {
	Path        string // absolute resolved path
	Original    string // existing file content (empty if new file)
	Proposed    string // proposed new content
	Diff        string // unified diff (Original -> Proposed)
	IsNewFile   bool
	GeneratedAt time.Time
}

// ProposeWrite creates a write proposal for a file without applying it.
// It enforces:
// - path is within root
// - no symlink escape
// - parent directory exists and is within root
// - no filesystem mutation
func ProposeWrite(g *Guard, path string, proposed string) (*WriteProposal, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return nil, err
	}

	var original string
	var isNew bool

	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			// New file proposal; ensure parent dir exists and is safe
			parent := filepath.Dir(resolved)
			if _, err := os.Stat(parent); err != nil {
				return nil, err
			}
			isNew = true
		} else {
			return nil, err
		}
	} else {
		if !info.Mode().IsRegular() {
			return nil, ErrPathOutsideRoot
		}

		b, err := os.ReadFile(resolved)
		if err != nil {
			return nil, err
		}
		original = string(b)
	}

	if !isNew && original == proposed {
		return nil, ErrNoChangeProposed
	}

	diff := unifiedDiff(
		original,
		proposed,
		displayPath(path),
		displayPath(path),
	)

	return &WriteProposal{
		Path:        resolved,
		Original:    original,
		Proposed:    proposed,
		Diff:        diff,
		IsNewFile:   isNew,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

// unifiedDiff produces a minimal unified diff without external deps.
// Line-based, deterministic, and stable for history views.
func unifiedDiff(oldText, newText, oldName, newName string) string {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	var buf bytes.Buffer
	buf.WriteString("--- " + oldName + "\n")
	buf.WriteString("+++ " + newName + "\n")

	// Simple, conservative diff: show full replacement when changes exist.
	// This is intentional for clarity and safety.
	buf.WriteString("@@\n")
	for _, l := range oldLines {
		buf.WriteString("-" + l + "\n")
	}
	for _, l := range newLines {
		buf.WriteString("+" + l + "\n")
	}

	return buf.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	// Preserve trailing newline semantics
	lines := strings.Split(s, "\n")
	if strings.HasSuffix(s, "\n") {
		return lines[:len(lines)-1]
	}
	return lines
}

// displayPath keeps diffs readable and repo-relative when possible.
func displayPath(p string) string {
	return filepath.ToSlash(p)
}

