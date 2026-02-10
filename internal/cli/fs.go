package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cshaiku/goshi/internal/app"
	"github.com/spf13/cobra"
)

func newFSCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fs",
		Short: "Local filesystem actions (safe, scoped, auditable)",
		Long: `Safely perform scoped filesystem operations with full auditability.

All operations are scoped to the repository root, auditable, and reversible.

QUICK START:
  $ goshi fs read src/main.go          # Read a file
  $ goshi fs list ./internal           # List a directory
  $ echo "content" | goshi fs write file.txt  # Propose a write
  $ goshi fs apply <id> --dry-run=false --yes # Apply proposal

SAFETY:
  - Filesystem jail: Cannot escape repository root
  - Symlink protection: Symlinks resolved safely
  - Proposal system: Review changes before applying
  - Permissions: User confirms sensitive operations

SEE ALSO:
  goshi help fs read    - Read a file or list directory recursively
  goshi help fs list    - List directory contents
  goshi help fs write   - Propose a file write (from stdin)
  goshi help fs apply   - Apply a write proposal (review first)
  goshi help fs probe   - Experimental: Test LLM filesystem handshake

ENVIRONMENT:
  GOSHI_CONFIG  - Path to configuration file (default: searched in standard locations)`,
	}

	cmd.AddCommand(
		newFSReadCommand(),
		newFSListCommand(),
		newFSWriteCommand(),
		newFSApplyCommand(),
		newFSProbeCmd(),
	)

	return cmd
}

func newFSReadCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <path>",
		Short: "Read a file or recursively list files in a directory",
		Long: `Read a file or generate a recursive file list.

MODES:
  - Regular file: Read contents of a single file
  - Directory: Automatically lists all files recursively as JSON

Output for directories is a JSON object with:
  - path: The root directory scanned
  - files: Array of relative file paths
  - count: Total number of files found

EXAMPLES:
  $ goshi fs read src/main.go
  Outputs file contents

  $ goshi fs read ./src
  Automatically scans recursively if src is a directory, outputs JSON:
  {
    "path": "/absolute/path/src",
    "files": ["main.go", "util/helper.go", "util/test.go"],
    "count": 3
  }

EXIT CODES:
  0   - Success: File read or directory list completed
  1   - Error: File/directory not found or permission denied`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := app.NewActionService(".")
			if err != nil {
				return err
			}

			path := args[0]

			// Check if the path is a directory and auto-enable recursive listing
			absPath, err := filepath.Abs(path)
			if err == nil {
				info, err := os.Stat(absPath)
				if err == nil && info.IsDir() {
					// Path is a directory, use recursive listing
					out, err := svc.RunAction("fs.list-recursive", map[string]any{
						"path": path,
					})
					if err != nil {
						return err
					}
					return printJSON(out)
				}
			}

			// Otherwise, read the single file
			out, err := svc.RunAction("fs.read", map[string]any{
				"path": path,
			})
			if err != nil {
				return err
			}

			return printJSON(out)
		},
	}

	return cmd
}

func newFSListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list [path]",
		Short: "List a directory safely",
		Long: `List directory contents safely within repository bounds.

Lists files and subdirectories for the specified path. By default,
lists the current directory. Output is JSON for easy parsing.

EXAMPLES:
  $ goshi fs list

  $ goshi fs list ./src

  $ goshi fs list | jq '.files'

EXIT CODES:
  0   - Success: Directory listed successfully
  1   - Error: Directory not found or access denied`,
		Args: cobra.RangeArgs(0, 1),
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
		Long: `Propose a file write operation with content from stdin.

Reads content from standard input and creates a reproducible write proposal.
The proposal includes a hash of the proposed changes and can be applied later
using 'goshi fs apply'.

EXAMPLES:
  $ echo "hello" | goshi fs write myfile.txt

  $ cat template.txt | goshi fs write out/generated.txt

  $ goshi fs write config.yaml < new-config.yml

EXIT CODES:
  0   - Success: Write proposal created
  1   - Error: No stdin provided or invalid path`,
		Args: cobra.ExactArgs(1),
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
