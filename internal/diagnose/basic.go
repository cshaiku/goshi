package diagnose

import "github.com/cshaiku/goshi/internal/detect"

type BasicDiagnoser struct{}

func (d *BasicDiagnoser) Diagnose(res detect.Result) (Result, error) {
	out := Result{
		Issues: []Issue{},
	}

  for _, bin := range res.MissingBinaries {
  	out.Issues = append(out.Issues, Issue{
  		Code:     "missing_binary",
  		Message:  "binary not found: " + bin,
  		Strategy: "install_" + bin,
  		Severity: SeverityError,
  	})
  }

  for _, w := range res.Warnings {
  	out.Issues = append(out.Issues, Issue{
  		Code:     "warning",
  		Message:  w,
  		Strategy: "manual_review",
  		Severity: SeverityWarn,
  	})
  }

	return out, nil
}
