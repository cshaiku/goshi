package repair

import "goshi/internal/diagnose"

type Repairer interface {
	Plan(diagnose.Result) (Plan, error)
}
