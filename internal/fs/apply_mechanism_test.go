package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cshaiku/goshi/internal/fs"
)

func TestApplyWriteWritesFileFromProposal(t *testing.T) {
	workspace := t.TempDir()

	oldwd, _ := os.Getwd()
	defer os.Chdir(oldwd)
	if err := os.Chdir(workspace); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	content := []byte("DATA")
	contentHash := fs.ComputeHash(content)

	p := fs.Proposal{
		ID:          fs.ProposalID("file.txt", true, "", contentHash),
		Path:        filepath.Join(workspace, "file.txt"),
		IsNewFile:   true,
		BaseHash:    "",
		ContentHash: contentHash,
	}

	if err := fs.SaveProposal(p); err != nil {
		t.Fatalf("SaveProposal failed: %v", err)
	}

	if err := fs.ApplyWriteProposal(p.ID); err != nil {
		t.Fatalf("ApplyWriteProposal failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(workspace, "file.txt")); err != nil {
		t.Fatalf("file not created")
	}
}
