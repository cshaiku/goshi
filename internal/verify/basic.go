package verify

import "grokgo/internal/detect"

type BasicVerifier struct {
	Binaries []string
}

func (v *BasicVerifier) Verify() (Result, error) {
	d := &detect.BasicDetector{
		Binaries: v.Binaries,
	}

	res, err := d.Detect()
	if err != nil {
		return Result{}, err
	}

	out := Result{
		Passed:   true,
		Failures: []string{},
	}

	if len(res.MissingBinaries) > 0 {
		out.Passed = false
		for _, b := range res.MissingBinaries {
			out.Failures = append(out.Failures, "missing binary: "+b)
		}
	}

	return out, nil
}
