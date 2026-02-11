package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Proposal struct {
	ID          string    `json:"id"`
	Path        string    `json:"path"`
	IsNewFile   bool      `json:"is_new_file"`
	BaseHash    string    `json:"base_hash"`
	ContentHash string    `json:"content_hash"`
	Diff        string    `json:"diff"`
	GeneratedAt time.Time `json:"generated_at"`
}

func ComputeHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func ProposalID(path string, isNew bool, baseHash, contentHash string) string {
	raw := path + "|" + boolToString(isNew) + "|" + baseHash + "|" + contentHash
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func SaveProposal(p Proposal) error {
	dir := filepath.Join(".goshi", "proposals")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, p.ID+".json"), data, 0644)
}

func LoadProposal(id string) (Proposal, error) {
	var p Proposal
	data, err := os.ReadFile(filepath.Join(".goshi", "proposals", id+".json"))
	if err != nil {
		return p, err
	}

	err = json.Unmarshal(data, &p)
	return p, err
}

func boolToString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
