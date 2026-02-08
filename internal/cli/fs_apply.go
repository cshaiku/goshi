package cli

import (
	"github.com/cshaiku/goshi/internal/fs"
	"github.com/spf13/cobra"
)

func newFSApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <proposal-id>",
		Short: "Apply a previously generated fs write proposal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Pass proposal ID directly.
			return fs.ApplyWriteProposal(args[0])
		},
	}
}
