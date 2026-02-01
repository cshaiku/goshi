package diagnose

import "github.com/cshaiku/goshi/internal/detect"

type Diagnoser interface {
	Diagnose(detect.Result) (Result, error)
}
