package detect

import (
	"os"
	"os/exec"
	"strings"
)

type BasicDetector struct {
	Binaries []string
}

func (d *BasicDetector) Detect() (Result, error) {
	res := Result{
		MissingBinaries: []string{},
		BrokenBinaries:  []string{},
		Warnings:        []string{},
	}

	path := os.Getenv("PATH")
	if path == "" {
		res.Warnings = append(res.Warnings, "PATH is empty")
	} else {
		parts := strings.Split(path, ":")
		if len(parts) == 0 {
			res.Warnings = append(res.Warnings, "PATH has no entries")
		}
	}

	for _, bin := range d.Binaries {
		_, err := exec.LookPath(bin)
		if err != nil {
			res.MissingBinaries = append(res.MissingBinaries, bin)
		}
	}

	return res, nil
}
