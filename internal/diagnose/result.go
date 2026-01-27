package diagnose

type Issue struct {
	Code     string
	Message  string
	Strategy string
}

type Result struct {
	Issues []Issue
}
