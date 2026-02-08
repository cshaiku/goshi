package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cshaiku/goshi/internal/app"
	"github.com/spf13/cobra"
)

func newFSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "Local filesystem actions (safe, scoped, auditable)",
	}

	cmd.AddCommand(
		newFSReadCommand(),
		newFSListCommand(),
		newFSWriteCommand(),
		newFSApplyCommand(),
	)

	return cmd
}

func newFSReadCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "read <path>",
		Short: "Read a file safely",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := app.NewActionService(".")
			if err != nil {
				return err
			}

			out, err := svc.RunAction("fs.read", map[string]any{
				"path": args[0],
			})
			if err != nil {
				return err
			}

			return printJSON(out)
		},
	}
}

func newFSListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [path]",
		Short: "List a directory safely",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}

			svc, err := app.NewActionService(".")
			if err != nil {
				return err
			}

			out, err := svc.RunAction("fs.list", map[string]any{
				"path": path,
			})
			if err != nil {
				return err
			}

			return printJSON(out)
		},
	}
}

func newFSWriteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "write <path>",
		Short: "Propose a write (content read from stdin)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
			if len(b) == 0 {
				return fmt.Errorf("no input on stdin (pipe content into fs write)")
			}

			svc, err := app.NewActionService(".")
			if err != nil {
				return err
			}

			out, err := svc.RunAction("fs.write", map[string]any{
			  "path":    args[0],
			  "content": string(b),
			})

			if err != nil {
			  return err
			}

			return printJSON(out)
		},
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
