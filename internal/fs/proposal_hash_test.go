package fs_test

import (
  "testing"

  "github.com/cshaiku/goshi/internal/fs"
)

func TestComputeHashDeterministic(t *testing.T) {
  a := fs.ComputeHash([]byte("data"))
  b := fs.ComputeHash([]byte("data"))

  if a != b {
    t.Fatalf("hashes must be deterministic")
  }
}
