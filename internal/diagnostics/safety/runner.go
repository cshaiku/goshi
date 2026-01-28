package safety

func Run() ([]InvariantResult, bool) {
  results := []InvariantResult{
    checkBinaryName(),
    checkRepoRootMarker(),
    checkCwdScope(),
    checkGitClean(),
    checkExecutionUser(),
  }

  safe := true
  for _, r := range results {
    if !r.Passed {
      safe = false
      break
    }
  }

  return results, safe
}
