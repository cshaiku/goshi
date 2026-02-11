package safety

import (
	"os"
	"path/filepath"
	"strings"
)

func checkCwdScope() InvariantResult {
	cwd, _ := os.Getwd()
	root, _ := filepath.Abs(".")

	ok := strings.HasPrefix(cwd, root)

	return InvariantResult{
		Name:     "working_dir_scope",
		Passed:   ok,
		Expected: "cwd within repo root",
		Actual:   cwd,
	}
}
