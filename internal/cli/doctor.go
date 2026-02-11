package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/diagnose"
	"github.com/cshaiku/goshi/internal/diagnostics/integrity"
	"github.com/cshaiku/goshi/internal/diagnostics/modules"
	"gopkg.in/yaml.v3"
)

func newDoctorCmd(cfg *config.Config) *cobra.Command {
	var format string
	var jsonCompat bool
	cmd := &cobra.Command{
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
  - Go module integrity
	- Source file integrity (via .goshi/goshi.manifest)

SEVERITY LEVELS:
  OK      - No issues detected
  WARN    - Warning-level issues (exit code 1)
  ERROR   - Error-level issues (exit code 2)
  FATAL   - Fatal issues (exit code 3)

FLAGS:
  --format=human  Output format: json, yaml, or human (default: human)
  --json          (DEPRECATED) Use --format=json instead

EXAMPLES:

  1. Check environment health (human-readable output):
     $ goshi doctor
     Shows a formatted list of any detected issues with descriptions.

  2. Check environment health (JSON output):
     $ goshi doctor --format=json
     Returns structured JSON with detected issues for automation.

  3. Check and capture results:
     $ goshi doctor > health_report.txt
     $ goshi doctor --format=json > health_report.json
     $ goshi doctor --format=yaml > health_report.yaml

EXIT CODES:
  0   - Healthy: No issues detected
  1   - Warning: Non-critical issues found
  2   - Error: Critical issues found
  3   - Fatal: System-level problems detected

ENVIRONMENT:
  GOSHI_CONFIG        - Path to configuration file to load
  GOSHI_MODEL         - LLM model to use (default: ollama)
  GOSHI_LLM_PROVIDER  - LLM provider: ollama, openai, etc. (default: auto)
  GOSHI_OLLAMA_URL    - Ollama server URL
  GOSHI_OLLAMA_PORT   - Ollama server port number

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

			// --- module diagnostics ---
			modDiag := modules.NewModuleDiagnostic()
			modIssues := modDiag.Run()
			diag.Issues = append(diag.Issues, modIssues...)

			// --- integrity diagnostics ---
			integrityDiag := integrity.NewIntegrityDiagnostic()
			integrityIssues := integrityDiag.Run()
			diag.Issues = append(diag.Issues, integrityIssues...)

			// Output format selection
			outFmt := format
			if outFmt == "" && jsonCompat {
				outFmt = "json"
			}
			switch outFmt {
			case "json":
				out, err := json.MarshalIndent(diag, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			case "yaml":
				data, err := yaml.Marshal(diag)
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			case "", "human":
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
			default:
				return fmt.Errorf("unknown format: %s (use 'json', 'yaml', or 'human')", outFmt)
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
	// Standardized output format flag
	cmd.Flags().StringVar(&format, "format", "", "Output format: json, yaml, or human (default: human)")
	cmd.Flags().BoolVar(&jsonCompat, "json", false, "(DEPRECATED) Output JSON (use --format=json)")
	return cmd
}
