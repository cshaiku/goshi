package fs

import (
	"os"
	"time"
)

// ProposeWrite creates a write proposal without mutating the filesystem.
func ProposeWrite(g *Guard, path string, proposed string) (WriteProposal, error) {
	resolved, err := g.Resolve(path)
	if err != nil {
		return WriteProposal{}, err
	}

	var baseHash string
	var isNew bool

	existing, err := os.ReadFile(resolved)
	if err != nil {
		if !os.IsNotExist(err) {
			return WriteProposal{}, err
		}
		isNew = true
	} else {
		baseHash = hashBytes(existing)
	}

	content := []byte(proposed)
	id := proposalID(resolved, isNew, baseHash, content)

	p := WriteProposal{
		ID:          id,
		Path:        resolved,
		IsNewFile:   isNew,
		BaseHash:    baseHash,
		Content:     content,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := saveProposal(p); err != nil {
		return WriteProposal{}, err
	}

	return p, nil
}
