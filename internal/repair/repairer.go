package repair

import "grokgo/internal/diagnose"

type Repairer interface {
	Plan(diagnose.Result) (Plan, error)
}
