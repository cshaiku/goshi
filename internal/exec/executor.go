package exec

import (
	"fmt"
	"os/exec"

	"grokgo/internal/repair"
)

type Executor struct {
	DryRun bool
}

func (e *Executor) Execute(plan repair.Plan) error {
	for _, a := range plan.Actions {
		if e.DryRun {
			fmt.Printf("[dry-run] would run: %v\n", a.Command)
			continue
		}

		fmt.Printf("[execute] running: %v\n", a.Command)

		cmd := exec.Command(a.Command[0], a.Command[1:]...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf(
				"command failed: %v\noutput:\n%s",
				a.Command,
				string(out),
			)
		}

		if len(out) > 0 {
			fmt.Printf("[output]\n%s\n", string(out))
		}
	}

	return nil
}
