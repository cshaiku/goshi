package diagnose

type Issue struct {
	Code     string
	Message  string
	Strategy string
  Severity Severity
}

type Result struct {
	Issues []Issue
}
