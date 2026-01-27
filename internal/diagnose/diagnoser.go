package diagnose

import "grokgo/internal/detect"

type Diagnoser interface {
	Diagnose(detect.Result) (Result, error)
}
