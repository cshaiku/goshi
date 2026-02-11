package safety

import (
	"os/exec"
	"strings"
)

func checkGitClean() InvariantResult {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	dirty := strings.TrimSpace(string(out)) != ""

	passed := err == nil && !dirty

	return InvariantResult{
		Name:     "git_clean_for_heal",
		Passed:   passed,
		Expected: "clean git working tree",
		Actual:   string(out),
	}
}
