package repair

import "goshi/internal/diagnose"

type BasicRepairer struct{}

func (r *BasicRepairer) Plan(diag diagnose.Result) (Plan, error) {
	out := Plan{
		Actions: []Action{},
	}

	for _, issue := range diag.Issues {
		switch issue.Code {
		case "missing_binary":
			out.Actions = append(out.Actions, Action{
				Code:        issue.Strategy,
				Description: "Install missing binary",
				Command:     []string{"apt", "install", "-y", issue.Strategy[len("install_"):]},
			})
		}
	}

	return out, nil
}
