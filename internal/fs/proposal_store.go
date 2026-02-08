package fs

import (
  "crypto/sha256"
  "encoding/hex"
  "encoding/json"
  "os"
  "path/filepath"
)

type WriteProposal struct {
  ID          string `json:"id"`
  Path        string `json:"path"`
  IsNewFile   bool   `json:"is_new_file"`
  BaseHash    string `json:"base_hash"`
  Content     []byte `json:"content"`
  Diff        string `json:"diff"`
  GeneratedAt string `json:"generated_at"`
}

func hashBytes(b []byte) string {
  sum := sha256.Sum256(b)
  return hex.EncodeToString(sum[:])
}

func proposalID(path string, isNew bool, baseHash string, content []byte) string {
  raw := path + "|" + baseHash + "|" + hashBytes(content)
  sum := sha256.Sum256([]byte(raw))
  return hex.EncodeToString(sum[:])
}

func proposalPath(id string) string {
  return filepath.Join(".goshi", "proposals", id+".json")
}

func saveProposal(p WriteProposal) error {
  if err := os.MkdirAll(filepath.Dir(proposalPath(p.ID)), 0755); err != nil {
    return err
  }

  data, err := json.MarshalIndent(p, "", "  ")
  if err != nil {
    return err
  }

  return os.WriteFile(proposalPath(p.ID), data, 0644)
}

func loadProposal(id string) (WriteProposal, error) {
  var p WriteProposal
  data, err := os.ReadFile(proposalPath(id))
  if err != nil {
    return p, err
  }
  err = json.Unmarshal(data, &p)
  return p, err
}
