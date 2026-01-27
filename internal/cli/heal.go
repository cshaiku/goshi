package cli

import (
	"fmt"

	"github.com/spf13/cobra"

  "grokgo/internal/config"
  "grokgo/internal/detect"
	"grokgo/internal/diagnose"
  "grokgo/internal/exec"
	"grokgo/internal/repair"
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
		Short: "Plan repairs for detected environment issues",
		RunE: func(cmd *cobra.Command, args []string) error {
      mode := "DRY-RUN"
      if !cfg.DryRun {
        mode = "EXECUTE"
      }

      fmt.Printf("Heal mode: %s\n", mode)

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

			dg := &diagnose.BasicDiagnoser{}
			diag, err := dg.Diagnose(res)
			if err != nil {
				return err
			}

			if len(diag.Issues) == 0 {
				fmt.Println("✔ nothing to repair")
				return nil
			}

			r := &repair.BasicRepairer{}
			plan, err := r.Plan(diag)
			if err != nil {
				return err
			}

			if len(plan.Actions) == 0 {
				fmt.Println("No repair actions available")
				return nil
			}

      if !cfg.DryRun {
      	fmt.Println("The following actions will be executed:")
      	for _, a := range plan.Actions {
      		fmt.Printf(" - %v\n", a.Command)
      	}

      	if !confirmExecution() {
      		fmt.Println("Aborted.")
      		return nil
      	}
      }

      ex := &exec.Executor{
      	DryRun: cfg.DryRun,
      }

      if err := ex.Execute(plan); err != nil {
      	return err
      }


			return nil
		},
	}
}
