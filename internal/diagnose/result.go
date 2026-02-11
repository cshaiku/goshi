package diagnose

type Issue struct {
	Code     string
	Message  string
	Strategy string
	Severity Severity
}

type Result struct {
	Version *VersionInfo `json:"version,omitempty" yaml:"version,omitempty"`
	Issues  []Issue
}

// VersionInfo captures the running goshi version for diagnostic output.
type VersionInfo struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	BuildTime string `json:"build_time" yaml:"build_time"`
	Dirty     string `json:"dirty" yaml:"dirty"`
}
