package fs

import (
	"os"
	"path/filepath"
)

// ListEntry describes a single directory entry.
type ListEntry struct {
	Name  string // base name
	Path  string // absolute resolved path
	IsDir bool
	Size  int64 // bytes (0 for directories)
}

// ListResult is the structured result of a directory listing.
type ListResult struct {
	Path    string      // absolute resolved directory path
	Entries []ListEntry // direct children only
}

// List lists the contents of a directory safely within the Guard root.
// It enforces:
// - path is within root
// - no symlink escape
// - target is a directory
// - non-recursive listing only
func List(g *Guard, path string) (*ListResult, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, ErrPathOutsideRoot
	}

	entries, err := os.ReadDir(resolved)
	if err != nil {
		return nil, err
	}

	result := &ListResult{
		Path: resolved,
	}

	for _, e := range entries {
		entryPath := filepath.Join(resolved, e.Name())

		info, err := e.Info()
		if err != nil {
			// Skip entries we can't stat safely
			continue
		}

		result.Entries = append(result.Entries, ListEntry{
			Name:  e.Name(),
			Path:  entryPath,
			IsDir: info.IsDir(),
			Size:  info.Size(),
		})
	}

	return result, nil
}
