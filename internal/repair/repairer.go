package repair

import "github.com/cshaiku/goshi/internal/diagnose"

type Repairer interface {
	Plan(diagnose.Result) (Plan, error)
}
