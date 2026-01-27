package detect

type Detector interface {
	Detect() (Result, error)
}
