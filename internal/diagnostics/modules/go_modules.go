package modules

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cshaiku/goshi/internal/diagnose"
)

// ModuleDiagnostic checks Go module health and security
type ModuleDiagnostic struct {
	CheckIntegrity       bool
	CheckUpdates         bool
	CheckVulnerabilities bool
	CheckTidy            bool
}

// Run executes all enabled module diagnostics
func (m *ModuleDiagnostic) Run() []diagnose.Issue {
	var issues []diagnose.Issue

	// 1. Verify go.sum integrity
	if m.CheckIntegrity {
		if err := m.verifyModules(); err != nil {
			issues = append(issues, diagnose.Issue{
				Severity: diagnose.SeverityError,
				Code:     "MOD_VERIFY_FAILED",
				Message:  fmt.Sprintf("Module integrity check failed: %v", err),
				Strategy: "Run 'go mod verify' and 'go mod tidy' to fix",
			})
		}
	}

	// 2. Check if go.mod and go.sum are tidy
	if m.CheckTidy {
		if err := m.checkTidy(); err != nil {
			issues = append(issues, diagnose.Issue{
				Severity: diagnose.SeverityWarn,
				Code:     "MOD_NOT_TIDY",
				Message:  "go.mod/go.sum may have unused or missing dependencies",
				Strategy: "Run 'go mod tidy' to clean up",
			})
		}
	}

	// 3. Check for outdated dependencies
	if m.CheckUpdates {
		outdated, err := m.checkOutdated()
		if err != nil {
			issues = append(issues, diagnose.Issue{
				Severity: diagnose.SeverityWarn,
				Code:     "MOD_UPDATE_CHECK_FAILED",
				Message:  fmt.Sprintf("Unable to check for updates: %v", err),
				Strategy: "Ensure network connectivity and try 'go list -u -m all'",
			})
		} else if len(outdated) > 0 {
			issues = append(issues, diagnose.Issue{
				Severity: diagnose.SeverityWarn,
				Code:     "MOD_OUTDATED",
				Message:  fmt.Sprintf("%d outdated dependencies found", len(outdated)),
				Strategy: "Run 'go get -u ./...' to update, or 'go list -u -m all' to review",
			})
		}
	}

	// 4. Check for known vulnerabilities (if govulncheck available)
	if m.CheckVulnerabilities {
		vulns, err := m.checkVulnerabilities()
		if err != nil {
			// govulncheck not installed or other error
			issues = append(issues, diagnose.Issue{
				Severity: diagnose.SeverityWarn,
				Code:     "VULN_CHECK_UNAVAILABLE",
				Message:  "govulncheck not available for vulnerability scanning",
				Strategy: "Install with 'go install golang.org/x/vuln/cmd/govulncheck@latest'",
			})
		} else if vulns > 0 {
			issues = append(issues, diagnose.Issue{
				Severity: diagnose.SeverityError,
				Code:     "VULN_DETECTED",
				Message:  fmt.Sprintf("Known vulnerabilities detected in dependencies"),
				Strategy: "Run 'govulncheck ./...' for details and update affected packages",
			})
		}
	}

	return issues
}

// verifyModules checks module integrity via go mod verify
func (m *ModuleDiagnostic) verifyModules() error {
	cmd := exec.Command("go", "mod", "verify")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, stderr.String())
	}
	return nil
}

// checkTidy verifies go.mod and go.sum are tidy
func (m *ModuleDiagnostic) checkTidy() error {
	// Run go mod tidy with -diff to see if changes would be made
	cmd := exec.Command("go", "mod", "tidy", "-diff")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// Exit code 0 means no changes needed
	if err := cmd.Run(); err != nil {
		// Non-zero means changes would be made
		return fmt.Errorf("go mod tidy would make changes")
	}
	// If stdout has content, changes would be made
	if stdout.Len() > 0 {
		return fmt.Errorf("go mod tidy would make changes")
	}
	return nil
}

// checkOutdated finds outdated module dependencies
func (m *ModuleDiagnostic) checkOutdated() ([]string, error) {
	cmd := exec.Command("go", "list", "-u", "-m", "all")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	var outdated []string
	lines := strings.Split(stdout.String(), "\n")
	for _, line := range lines {
		// Lines with available updates contain " ["
		// Example: github.com/foo/bar v1.0.0 [v1.1.0]
		if strings.Contains(line, " [") && strings.Contains(line, "]") {
			outdated = append(outdated, line)
		}
	}
	return outdated, nil
}

// checkVulnerabilities runs govulncheck if available
func (m *ModuleDiagnostic) checkVulnerabilities() (int, error) {
	// Check if govulncheck is installed
	if _, err := exec.LookPath("govulncheck"); err != nil {
		return 0, err
	}
	cmd := exec.Command("govulncheck", "-json", "./...")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	// govulncheck exits with 3 if vulnerabilities found
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 3 {
			// Parse JSON output to count vulnerabilities
			// For now, just return that vulns were found
			return 1, nil
		}
	}
	// No vulnerabilities or other error
	if err != nil && err.Error() != "exit status 3" {
		return 0, err
	}
	return 0, nil
}

// NewModuleDiagnostic creates a diagnostic with all checks enabled
func NewModuleDiagnostic() *ModuleDiagnostic {
	return &ModuleDiagnostic{
		CheckIntegrity:       true,
		CheckUpdates:         true,
		CheckVulnerabilities: true,
		CheckTidy:            true,
	}
}
