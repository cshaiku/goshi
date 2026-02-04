package fs

import (
	"os"
)

// ReadResult is the structured result of a safe file read.
type ReadResult struct {
	Path    string // absolute resolved path
	Content string // full file contents
	Size    int64  // bytes
}

// Read reads a file safely within the Guard root.
// It enforces:
// - path is within root
// - no symlink escape
// - target is a regular file
func Read(g *Guard, path string) (*ReadResult, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}

	if !info.Mode().IsRegular() {
		return nil, ErrPathOutsideRoot
	}

	b, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}

	return &ReadResult{
		Path:    resolved,
		Content: string(b),
		Size:    info.Size(),
	}, nil
}
