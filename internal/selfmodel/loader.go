package selfmodel

import (
  "fmt"
  "os"
)

const DefaultPath = "goshi.self.model.yaml"

type SelfModel struct {
  Path string
  Raw  string
}

func Load(path string) (*SelfModel, error) {
  if path == "" {
    path = DefaultPath
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
