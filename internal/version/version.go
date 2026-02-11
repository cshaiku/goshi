package version

import "fmt"

// Version information set at build time via -ldflags.
var (
	Version   = "1.5.0"
	Commit    = "dev"
	BuildTime = "unknown"
	Dirty     = "false"
)

// Info describes the current build version.
type Info struct {
	Version   string
	Commit    string
	BuildTime string
	Dirty     string
}

// Current returns the current build version information.
func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildTime: BuildTime,
		Dirty:     Dirty,
	}
}

// String returns a concise version string.
func String() string {
	return fmt.Sprintf("%s (commit %s, built %s, dirty=%s)", Version, Commit, BuildTime, Dirty)
}
