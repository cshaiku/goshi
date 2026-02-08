package fs

import (
	"errors"
	"os"
)

var ErrDriftDetected = errors.New("drift detected: file has changed since proposal")

func ApplyWriteProposal(id string) error {
	p, err := loadProposal(id)
	if err != nil {
		return err
	}

	if !p.IsNewFile {
		current, err := os.ReadFile(p.Path)
		if err != nil {
			return err
		}

		if hashBytes(current) != p.BaseHash {
			return ErrDriftDetected
		}
	}

	return os.WriteFile(p.Path, p.Content, 0644)
}
