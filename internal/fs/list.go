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

// RecursiveListResult is the structured result of a recursive directory listing.
type RecursiveListResult struct {
	Path  string   // absolute resolved root directory path
	Files []string // relative paths to all files found
	Count int      // total number of files
}

// ListRecursive recursively lists all files in a directory tree safely within the Guard root.
// It enforces:
// - path is within root
// - no symlink escape
// - target is a directory
// - returns only regular files (no directories)
// - returns relative paths from the root directory
func ListRecursive(g *Guard, path string) (*RecursiveListResult, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, ErrPathNotDir
	}

	result := &RecursiveListResult{
		Path:  resolved,
		Files: []string{},
	}

	// Walk the directory tree
	err = filepath.Walk(resolved, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access
			return nil
		}

		// Only include regular files, skip directories
		if !info.IsDir() {
			// Calculate relative path from the root
			relPath, err := filepath.Rel(resolved, filePath)
			if err != nil {
				return nil
			}

			result.Files = append(result.Files, relPath)
			result.Count++
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
