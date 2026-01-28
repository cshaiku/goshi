package safety

import (
  "os"
  "path/filepath"
)

func checkBinaryName() InvariantResult {
  actual := filepath.Base(os.Args[0])
  expected := "grokgo"

  return InvariantResult{
    Name:     "binary_name",
    Passed:   actual == expected,
    Expected: expected,
    Actual:   actual,
  }
}
