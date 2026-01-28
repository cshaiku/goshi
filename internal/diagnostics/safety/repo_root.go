package safety

import "os"

func checkRepoRootMarker() InvariantResult {
  marker := "grokgo.self.model.yaml"
  _, err := os.Stat(marker)

  return InvariantResult{
    Name:     "repo_root_marker",
    Passed:   err == nil,
    Expected: "file exists",
    Actual:   errString(err),
  }
}
