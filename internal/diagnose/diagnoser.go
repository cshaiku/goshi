package diagnose

import "goshi/internal/detect"

type Diagnoser interface {
	Diagnose(detect.Result) (Result, error)
}
