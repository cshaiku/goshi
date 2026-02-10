package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/detect"
	"github.com/cshaiku/goshi/internal/diagnose"
	"github.com/cshaiku/goshi/internal/exec"
	"github.com/cshaiku/goshi/internal/repair"
	"github.com/cshaiku/goshi/internal/verify"
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
	return &cobra.Command{
		Use:   "heal",
		Short: "Repair detected environment issues",
		Long: `Analyze your environment and automatically repair identified issues.

DESCRIPTION:
This command performs a multi-stage diagnostic and repair workflow:
  1. Detect      - Identify missing or misconfigured binaries (git, curl, jq)
  2. Diagnose    - Analyze detected problems and assess severity
  3. Plan        - Generate repair actions based on diagnosis
  4. Execute     - Run repair commands (requires confirmation)
  5. Verify      - Confirm repairs were successful

WORKFLOW SAFETY:
By default, heal runs in DRY-RUN mode, which shows what would be done without
making any changes. You must explicitly disable dry-run mode to actually execute
repairs. Confirmation is required before execution (unless --yes is given).

FLAGS:
  --dry-run=false    Execute repairs (default: true for safety)
  --yes              Skip confirmation prompts and proceed automatically
  --json             Output machine-readable JSON instead of human-friendly text

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
     $ goshi heal --json
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
  GOSHI_MODEL         - LLM model to use (default: ollama)
  GOSHI_LLM_PROVIDER  - LLM provider (default: auto)

SEE ALSO:
  goshi doctor        - Check environment health without repairing
  goshi help          - Show general help information`,
		RunE: func(cmd *cobra.Command, args []string) error {

			// --- JSON mode (machine output only) ---
			if cfg.JSON {
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
			}

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

			if len(diag.Issues) == 0 {
				fmt.Println("✔ nothing to repair")
				return nil
			}

			// --- plan ---
			r := &repair.BasicRepairer{}
			plan, err := r.Plan(diag)
			if err != nil {
				return err
			}

			if len(plan.Actions) == 0 {
				fmt.Println("No repair actions available")
				return nil
			}

			// --- confirmation gate ---
			if !cfg.DryRun {
				fmt.Println("The following actions will be executed:")
				for _, a := range plan.Actions {
					fmt.Printf(" - %v\n", a.Command)
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
		},
	}
}
