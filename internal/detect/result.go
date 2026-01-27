package detect

type Result struct {
	MissingBinaries []string
	BrokenBinaries  []string
	Warnings        []string
}
