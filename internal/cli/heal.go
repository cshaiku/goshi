package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/diagnose"
	"github.com/cshaiku/goshi/internal/diagnostics/integrity"
	"github.com/cshaiku/goshi/internal/exec"
	"github.com/cshaiku/goshi/internal/repair"
	"github.com/cshaiku/goshi/internal/verify"
	"gopkg.in/yaml.v3"
)

func confirmExecution() bool {
	fmt.Print("Proceed with execution? Type 'yes' to continue: ")

	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false
	}

	return input == "yes"
}

func newHealCmd(cfg *config.Config) *cobra.Command {
	var format string
	var jsonCompat bool
	var dryRun bool
	var yes bool
	cmd := &cobra.Command{
		Use:   "heal",
		Short: "Repair detected environment issues",
		Long: `Analyze your environment and automatically repair identified issues.

DESCRIPTION:
This command performs a multi-stage diagnostic and repair workflow:
  1. Detect      - Identify missing or misconfigured binaries (git, curl, jq)
  2. Diagnose    - Analyze detected problems and assess severity
	3. Integrity   - Validate source reference bundle and plan restore actions
	4. Plan        - Generate repair actions based on diagnosis
	5. Execute     - Run repair commands (requires confirmation)
	6. Verify      - Confirm repairs were successful

WORKFLOW SAFETY:
By default, heal runs in DRY-RUN mode, which shows what would be done without
making any changes. You must explicitly disable dry-run mode to actually execute
repairs. Confirmation is required before execution (unless --yes is given).

FLAGS:
  --dry-run=true      Run in dry-run mode (default: true for safety)
  --yes               Skip confirmation prompts and proceed automatically
  --format=human      Output format: json, yaml, or human (default: human)
  --json              (DEPRECATED) Use --format=json instead

EXAMPLES:

  1. Preview what would be repaired (dry-run):
     $ goshi heal
     This is the safest default - shows proposed repairs without executing them.

  2. Execute repairs with confirmation prompt:
     $ goshi heal --dry-run=false
     Will show planned actions and ask for confirmation before executing.

  3. Execute repairs automatically without prompting:
     $ goshi heal --dry-run=false --yes
     Use with caution - automatically repairs found issues without asking.

  4. Get machine-readable output for automation:
     $ goshi heal --format=json
     Returns JSON output showing the mode and status.

  5. Full pipeline: preview, then execute:
     $ goshi heal                           # First, see what needs fixing
     $ goshi heal --dry-run=false --yes    # Then execute with confirmation skipped

EXIT CODES:
  0   - Success (either in dry-run or all repairs passed verification)
  1   - Warning (issues found but warning-level)
  2   - Error (repair failed or verification failed)
  3   - Fatal (execution error during repairs)

ENVIRONMENT:
  GOSHI_CONFIG        - Path to configuration file to load
  GOSHI_MODEL         - LLM model to use (default: ollama)
  GOSHI_LLM_PROVIDER  - LLM provider: ollama, openai, etc. (default: auto)
  GOSHI_OLLAMA_URL    - Ollama server URL
  GOSHI_OLLAMA_PORT   - Ollama server port number

SEE ALSO:
  goshi doctor        - Check environment health without repairing
  goshi help          - Show general help information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.DryRun = dryRun
			cfg.Yes = yes
			outFmt := format
			if outFmt == "" && jsonCompat {
				outFmt = "json"
			}
			switch outFmt {
			case "json":
				mode := "execute"
				if cfg.DryRun {
					mode = "dry-run"
				}
				out, err := json.MarshalIndent(map[string]any{
					"mode": mode,
				}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(out))
				return nil
			case "yaml":
				mode := "execute"
				if cfg.DryRun {
					mode = "dry-run"
				}
				data, err := yaml.Marshal(map[string]any{
					"mode": mode,
				})
				if err != nil {
					return err
				}
				fmt.Print(string(data))
				return nil
			case "", "human":
				// --- human banner ---
				mode := "EXECUTE"
				if cfg.DryRun {
					mode = "DRY-RUN"
				}
				fmt.Printf("Heal mode: %s\n", mode)

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

				// --- integrity diagnostics ---
				integrityDiag := integrity.NewIntegrityDiagnostic()
				manifest, integrityResult, integrityErr := integrityDiag.PlanRepair()
				integrityTargets := make([]string, 0)
				if integrityErr == nil {
					seen := make(map[string]struct{})
					for _, path := range integrityResult.MissingFiles {
						seen[path] = struct{}{}
					}
					for _, mod := range integrityResult.ModifiedFiles {
						seen[mod.Path] = struct{}{}
					}
					for path := range seen {
						integrityTargets = append(integrityTargets, path)
					}
					sort.Strings(integrityTargets)
				}

				if len(diag.Issues) == 0 && len(integrityTargets) == 0 {
					if integrityErr != nil {
						fmt.Printf("✔ nothing to repair (integrity check unavailable: %v)\n", integrityErr)
						return nil
					}
					fmt.Println("✔ nothing to repair")
					return nil
				}

				// --- plan ---
				r := &repair.BasicRepairer{}
				plan, err := r.Plan(diag)
				if err != nil {
					return err
				}

				if len(plan.Actions) == 0 && len(integrityTargets) == 0 {
					fmt.Println("No repair actions available")
					return nil
				}

				if integrityErr != nil {
					fmt.Printf("Integrity repair unavailable: %v\n", integrityErr)
				} else if len(integrityTargets) > 0 {
					if cfg.DryRun {
						fmt.Println("Integrity restore plan (dry-run):")
					} else {
						fmt.Println("Integrity restore plan:")
					}
					for _, path := range integrityTargets {
						fmt.Printf(" - restore %s\n", path)
					}
				}

				// --- confirmation gate ---
				if !cfg.DryRun {
					fmt.Println("The following actions will be executed:")
					for _, a := range plan.Actions {
						fmt.Printf(" - %v\n", a.Command)
					}
					for _, path := range integrityTargets {
						fmt.Printf(" - restore %s (from source tarball)\n", path)
					}

					if !cfg.Yes {
						if !confirmExecution() {
							fmt.Println("Aborted.")
							return nil
						}
					}
				}

				// --- execute ---
				ex := &exec.Executor{
					DryRun: cfg.DryRun,
				}

				if err := ex.Execute(plan); err != nil {
					// execution failure = fatal
					os.Exit(3)
				}

				if !cfg.DryRun && integrityErr == nil && len(integrityTargets) > 0 {
					restored, err := integrityDiag.RestoreFromTarball(manifest, integrityTargets)
					if err != nil {
						fmt.Printf("✖ integrity restore failed: %v\n", err)
						os.Exit(3)
					}
					fmt.Printf("✔ restored %d source files from tarball\n", len(restored))
				}

				// --- verify ---
				v := &verify.BasicVerifier{
					Binaries: []string{
						"git",
						"curl",
						"jq",
					},
				}

				vr, err := v.Verify()
				if err != nil {
					return err
				}

				if vr.Passed {
					fmt.Println("✔ verification passed")
				} else {
					fmt.Println("✖ verification failed:")
					for _, f := range vr.Failures {
						fmt.Println(" -", f)
					}
					os.Exit(2)
				}

				return nil
			default:
				return fmt.Errorf("unknown format: %s (use 'json', 'yaml', or 'human')", outFmt)
			}
		},
	}
	// Standardized output format flag
	cmd.Flags().StringVar(&format, "format", "", "Output format: json, yaml, or human (default: human)")
	cmd.Flags().BoolVar(&jsonCompat, "json", false, "(DEPRECATED) Output JSON (use --format=json)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "Run in dry-run mode (default: true)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompts")
	return cmd
}
