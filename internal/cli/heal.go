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
