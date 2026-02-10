package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/diagnose"
)

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check environment health",
		Long: `Diagnose environment health and identify issues.

DESCRIPTION:
This command analyzes your environment for configuration and dependency problems
without making any changes. It checks for the availability and configuration of
critical tools and reports any issues found, along with their severity levels.

CHECKS PERFORMED:
  - Binary availability (git, curl, jq)
  - Basic configuration validation
  - Environmental dependencies

SEVERITY LEVELS:
  OK      - No issues detected
  WARN    - Warning-level issues (exit code 1)
  ERROR   - Error-level issues (exit code 2)
  FATAL   - Fatal issues (exit code 3)

ENVIRONMENT VARIABLES:
  GOSHI_JSON              - Output JSON format (same as --json flag)
  GOSHI_MODEL             - LLM model to use (default: ollama)
  GOSHI_LLM_PROVIDER      - LLM provider (default: auto)

EXAMPLES:

  1. Check environment health (human-readable output):
     $ goshi doctor
     Shows a formatted list of any detected issues with descriptions.

  2. Check environment health (JSON output):
     $ goshi doctor --json
     Returns structured JSON with detected issues for automation.

  3. Check and capture results:
     $ goshi doctor > health_report.txt
     $ goshi doctor --json > health_report.json

OUTPUT FORMAT:

  Human format shows:
    [severity][code] message (recommended fix strategy)

  JSON format includes:
    - Issues array with code, severity, message, and strategy
    - Allows for programmatic parsing and automation

EXIT CODES:
  0   - Healthy: No issues detected
  1   - Warning: Non-critical issues found
  2   - Error: Critical issues found
  3   - Fatal: System-level problems detected

SEE ALSO:
  goshi heal              - Automatically repair detected issues
  goshi help              - Show general help information`,
		RunE: func(cmd *cobra.Command, args []string) error {

			// --- detect ---
			d := &detect.BasicDetector{
				Binaries: []string{
					"git",
					"curl",
					"jq",
				},
			}

			res, err := d.Detect()
			if err != nil {
				return err
			}

			// --- diagnose ---
			dg := &diagnose.BasicDiagnoser{}
			diag, err := dg.Diagnose(res)
			if err != nil {
				return err
			}

			// --- JSON output ---
			if cfg.JSON {
				out, err := json.MarshalIndent(diag, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			}

			// --- human output ---
			if len(diag.Issues) == 0 {
				fmt.Println("✔ environment looks healthy")
				return nil
			}

			fmt.Println("Detected issues:")
			for _, issue := range diag.Issues {
				fmt.Printf(
					" - [%s][%s] %s (suggested: %s)\n",
					issue.Severity,
					issue.Code,
					issue.Message,
					issue.Strategy,
				)
			}

			// --- exit code mapping ---
			overall := diagnose.AggregateSeverity(diag.Issues)

			switch overall {
			case diagnose.SeverityOK:
				return nil
			case diagnose.SeverityWarn:
				os.Exit(1)
			case diagnose.SeverityError:
				os.Exit(2)
			case diagnose.SeverityFatal:
				os.Exit(3)
			default:
				os.Exit(2)
			}

			return nil
		},
	}
}
