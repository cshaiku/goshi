package selfmodel

import (
	"fmt"
	"os"
	"path/filepath"
)

const DefaultPath = "goshi.self.model.yaml"

type SelfModel struct {
	Path string
	Raw  string
}

func Load(path string) (*SelfModel, error) {
	if path == "" {
		path = defaultModelPath()
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load self-model file %q: %w", path, err)
	}

	if len(b) == 0 {
		return nil, fmt.Errorf("self-model file %q is empty", path)
	}

	return &SelfModel{
		Path: path,
		Raw:  string(b),
	}, nil
}

// defaultModelPath preserves a project's model while allowing an installed CLI
// to start outside its source checkout. Explicit Load paths never fall back.
func defaultModelPath() string {
	executable, _ := os.Executable()
	return resolveModelPath(DefaultPath, executable)
}

// Only a missing project model may fall back to the installed resource.
func resolveModelPath(localPath, executable string) string {
	if _, err := os.Stat(localPath); !os.IsNotExist(err) {
		return localPath
	}
	if executable == "" {
		return localPath
	}
	candidate := filepath.Join(filepath.Dir(executable), "..", "share", "goshi", DefaultPath)
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return localPath
}
