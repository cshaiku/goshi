package safety

type InvariantResult struct {
	Name     string `json:"invariant"`
	Passed   bool   `json:"passed"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
}
