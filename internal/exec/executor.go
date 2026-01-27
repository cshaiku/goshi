package exec

import (
	"fmt"

	"grokgo/internal/repair"
)

type Executor struct {
	DryRun bool
}

func (e *Executor) Execute(plan repair.Plan) error {
	if len(plan.Actions) == 0 {
		return nil
	}

	for _, a := range plan.Actions {
		if e.DryRun {
			fmt.Printf("[dry-run] would run: %v\n", a.Command)
			continue
		}

		// REAL EXECUTION WILL LIVE HERE (later)
		fmt.Printf("[execute] running: %v\n", a.Command)
	}

	return nil
}
