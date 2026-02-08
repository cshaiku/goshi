package fs_test

import (
  "os"
  "path/filepath"
  "testing"

  "github.com/cshaiku/goshi/internal/fs"
)

func TestSaveProposalDoesNotWriteTargetFile(t *testing.T) {
  workspace := t.TempDir()

  oldwd, _ := os.Getwd()
  defer os.Chdir(oldwd)
  if err := os.Chdir(workspace); err != nil {
    t.Fatalf("chdir failed: %v", err)
  }

  content := []byte("HELLO")
  contentHash := fs.ComputeHash(content)

  p := fs.Proposal{
    ID:          fs.ProposalID("test.txt", true, "", contentHash),
    Path:        filepath.Join(workspace, "test.txt"),
    IsNewFile:   true,
    BaseHash:    "",
    ContentHash: contentHash,
  }

  if err := fs.SaveProposal(p); err != nil {
    t.Fatalf("SaveProposal failed: %v", err)
  }

  if _, err := os.Stat(filepath.Join(workspace, "test.txt")); !os.IsNotExist(err) {
    t.Fatalf("target file should not exist")
  }

  if _, err := os.Stat(filepath.Join(workspace, ".goshi", "proposals", p.ID+".json")); err != nil {
    t.Fatalf("proposal file not created")
  }
}
