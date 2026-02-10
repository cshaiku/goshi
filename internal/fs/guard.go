package fs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrPathOutsideRoot = errors.New("path resolves outside allowed root")
	ErrSymlinkEscape   = errors.New("path contains symlink that escapes root")
	ErrPathNotDir      = errors.New("path is not a directory")
)

// Guard enforces filesystem safety for local operations.
type Guard struct {
	root string // absolute, resolved
}

// NewGuard creates a Guard rooted at the given directory.
// The root is resolved to an absolute, symlink-free path.
func NewGuard(root string) (*Guard, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}

	return &Guard{root: real}, nil
}

// Resolve validates a user-supplied path and returns a safe absolute target path.
// It allows new files while preventing traversal or symlink escape.
func (g *Guard) Resolve(p string) (string, error) {
	if p == "" {
		return "", ErrPathOutsideRoot
	}

	clean := filepath.Clean(p)
	if strings.HasPrefix(clean, "..") {
		return "", ErrPathOutsideRoot
	}

	target := filepath.Join(g.root, clean)

	// Walk upward until we find an existing parent to validate symlinks
	parent := target
	for {
		info, err := os.Lstat(parent)
		if err == nil {
			// Existing path found; validate symlink resolution
			if info.Mode()&os.ModeSymlink != 0 {
				real, err := filepath.EvalSymlinks(parent)
				if err != nil {
					return "", err
				}
				if !isWithinRoot(g.root, real) {
					return "", ErrSymlinkEscape
				}
			}
			break
		}

		if os.IsNotExist(err) {
			next := filepath.Dir(parent)
			if next == parent {
				break
			}
			parent = next
			continue
		}

		return "", err
	}

	// Final target must still be within root
	if !isWithinRoot(g.root, target) {
		return "", ErrPathOutsideRoot
	}

	return target, nil
}

func isWithinRoot(root, p string) bool {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}
