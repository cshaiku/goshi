package selfmodel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstalledModelFallback(t *testing.T) {
	root := t.TempDir()
	exe := filepath.Join(root, "bin", "goshi")
	installed := filepath.Join(root, "share", "goshi", DefaultPath)
	if err := os.MkdirAll(filepath.Dir(installed), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installed, []byte("installed"), 0644); err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(root, "project", DefaultPath)
	if got := resolveModelPath(local, exe); got != installed {
		t.Fatalf("fallback: %s", got)
	}
	if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("project"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := resolveModelPath(local, exe); got != local {
		t.Fatalf("project model lost precedence: %s", got)
	}
	// An existing empty/corrupt project model must not silently select another policy.
	if err := os.WriteFile(local, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if got := resolveModelPath(local, exe); got != local {
		t.Fatal("empty project model fell back")
	}
	if _, err := Load(local); err == nil {
		t.Fatal("empty model accepted")
	}
	if _, err := Load(filepath.Join(root, "explicit-missing")); err == nil {
		t.Fatal("explicit missing model accepted")
	}
}
