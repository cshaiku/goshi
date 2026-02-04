package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/cshaiku/goshi/internal/fs"
	"github.com/spf13/cobra"
)

func newFSApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <proposal.json>",
		Short: "Apply a previously generated fs.write proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
		  b, err := os.ReadFile(args[0])
		  if err != nil {
		    return err
		  }

		  var raw map[string]any
		  if err := json.Unmarshal(b, &raw); err != nil {
		    return err
		  }

		  prop, err := proposalFromJSON(raw)
		  if err != nil {
		    return err
		  }

		  // Respect global --dry-run flag
		  dryRun, _ := cmd.Flags().GetBool("dry-run")
		  if dryRun {
		    fmt.Println("dry-run enabled; no changes applied")
		    fmt.Println(prop.Diff)
		    return nil
		  }

		  guard, err := fs.NewGuard(".")
		  if err != nil {
		    return err
		  }

		  if err := fs.ApplyWrite(guard, prop); err != nil {
		    return err
		  }

		  fmt.Println("applied successfully")
		  return nil
		},
	}
}

func proposalFromJSON(m map[string]any) (*fs.WriteProposal, error) {
	path, ok := m["path"].(string)
	if !ok {
		return nil, errors.New("invalid proposal: missing path")
	}

	diff, ok := m["diff"].(string)
	if !ok {
		return nil, errors.New("invalid proposal: missing diff")
	}

	isNew, _ := m["is_new_file"].(bool)

	// content is not required for apply; diff is authoritative
	return &fs.WriteProposal{
		Path:      path,
		Diff:      diff,
		IsNewFile: isNew,
	}, nil
}
