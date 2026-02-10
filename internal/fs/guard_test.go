package fs

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewGuardValid tests creating a Guard with a valid directory
func TestNewGuardValid(t *testing.T) {
	tmpDir := t.TempDir()

	guard, err := NewGuard(tmpDir)
	if err != nil {
		t.Errorf("expected valid directory to create guard, got error: %v", err)
	}

	if guard == nil {
		t.Errorf("expected non-nil guard")
	}

	if guard.root == "" {
		t.Errorf("expected guard root to be set")
	}
}

// TestGuardResolveSimplePath tests resolving a simple file path
func TestGuardResolveSimplePath(t *testing.T) {
	tmpDir := t.TempDir()
	guard, _ := NewGuard(tmpDir)

	resolved, err := guard.Resolve("test.txt")
	if err != nil {
		t.Errorf("expected simple path to resolve, got error: %v", err)
	}

	expected := filepath.Join(tmpDir, "test.txt")
	// Normalize paths for comparison (handles /private prefix on macOS)
	resolvedNorm, _ := filepath.EvalSymlinks(resolved)
	expectedNorm, _ := filepath.EvalSymlinks(expected)

	if resolvedNorm != expectedNorm {
		t.Errorf("expected resolved path %s, got %s", expectedNorm, resolvedNorm)
	}
}

// TestGuardResolveNestedPath tests resolving a nested file path
func TestGuardResolveNestedPath(t *testing.T) {
	tmpDir := t.TempDir()
	guard, _ := NewGuard(tmpDir)

	// Create nested directories
	os.MkdirAll(filepath.Join(tmpDir, "dir1", "dir2"), 0755)

	resolved, err := guard.Resolve("dir1/dir2/file.txt")
	if err != nil {
		t.Errorf("expected nested path to resolve, got error: %v", err)
	}

	expected := filepath.Join(tmpDir, "dir1", "dir2", "file.txt")
	// Normalize paths for comparison (handles /private prefix on macOS)
	resolvedNorm, _ := filepath.EvalSymlinks(resolved)
	expectedNorm, _ := filepath.EvalSymlinks(expected)

	if resolvedNorm != expectedNorm {
		t.Errorf("expected resolved path %s, got %s", expectedNorm, resolvedNorm)
	}
}

// TestGuardResolvePathTraversal tests that path traversal attacks are blocked
func TestGuardResolvePathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	guard, _ := NewGuard(tmpDir)

	tests := []string{
		"../etc/passwd",
		"../../etc/passwd",
		"../../../etc/passwd",
		"dir/../../etc/passwd",
	}

	for _, testPath := range tests {
		resolved, err := guard.Resolve(testPath)
		if err != ErrPathOutsideRoot {
			t.Errorf("expected path traversal %s to be blocked, got resolved: %s, error: %v", testPath, resolved, err)
		}
	}
}

// TestGuardResolveEmptyPath tests resolving an empty path
func TestGuardResolveEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	guard, _ := NewGuard(tmpDir)

	resolved, err := guard.Resolve("")
	if err != ErrPathOutsideRoot {
		t.Errorf("expected empty path to return error, got resolved: %s, error: %v", resolved, err)
	}
}

// TestGuardResolveDotPath tests resolving dot path
func TestGuardResolveDotPath(t *testing.T) {
	tmpDir := t.TempDir()
	guard, _ := NewGuard(tmpDir)

	resolved, err := guard.Resolve(".")
	if err != nil {
		t.Errorf("expected dot path to resolve, got error: %v", err)
	}

	// Normalize paths for comparison (handles /private prefix on macOS)
	resolvedNorm, _ := filepath.EvalSymlinks(resolved)
	tmpDirNorm, _ := filepath.EvalSymlinks(tmpDir)

	if resolvedNorm != tmpDirNorm {
		t.Errorf("expected dot path to resolve to root %s, got %s", tmpDirNorm, resolvedNorm)
	}
}

// TestGuardResolveCreatesNewFile tests that new files can be resolved
func TestGuardResolveCreatesNewFile(t *testing.T) {
	tmpDir := t.TempDir()
	guard, _ := NewGuard(tmpDir)

	newPath := "newfile.txt"
	resolved, err := guard.Resolve(newPath)
	if err != nil {
		t.Errorf("expected new file to be resolvable, got error: %v", err)
	}

	expected := filepath.Join(tmpDir, newPath)
	// Normalize paths for comparison (handles /private prefix on macOS)
	resolvedNorm, _ := filepath.EvalSymlinks(resolved)
	expectedNorm, _ := filepath.EvalSymlinks(expected)

	if resolvedNorm != expectedNorm {
		t.Errorf("expected resolved path %s, got %s", expectedNorm, resolvedNorm)
	}
}

// TestIsWithinRoot tests the isWithinRoot helper function
func TestIsWithinRoot(t *testing.T) {
	root := "/home/user/project"

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"same path", root, true},
		{"child path", filepath.Join(root, "file.txt"), true},
		{"nested child", filepath.Join(root, "dir1", "dir2", "file.txt"), true},
		{"parent path", "/home/user", false},
		{"sibling path", "/home/user/other", false},
		{"unrelated path", "/etc/passwd", false},
		{"root dot notation", filepath.Join(root, "."), true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := isWithinRoot(root, test.path)
			if result != test.expected {
				t.Errorf("isWithinRoot(%s, %s) expected %v, got %v", root, test.path, test.expected, result)
			}
		})
	}
}
