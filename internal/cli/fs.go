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

DESCRIPTION:
Provides safe, scoped methods for reading, listing, writing, and applying
changes to the local filesystem. All operations are:
  - Scoped to the repository root (cannot escape via symlinks)
  - Auditable (all operations are logged and can be reviewed)
  - Reversible (write operations are stored as proposals for review)

SUBCOMMANDS:
  read      - Read a file from disk
  list      - List contents of a directory
  write     - Create or update a file (stores as proposal)
  apply     - Apply a previously generated write proposal

SAFETY FEATURES:
  - Filesystem jail: Cannot read/write outside repository root
  - Symlink protection: Symlinks are resolved safely
  - Proposal system: Writes are proposed first, then applied manually
  - Permission prompts: User confirms sensitive operations

EXAMPLES:

  1. Read a file:
     $ goshi fs read src/main.go
     Outputs the file contents.

  2. List a directory:
     $ goshi fs list
     $ goshi fs list ./internal
     Lists directory contents (default: current directory).

  3. Propose a file write:
     $ echo "new content" | goshi fs write config.txt
     Creates a proposal for writing. Returns a proposal ID.

  4. Apply a write proposal:
     $ goshi fs apply <proposal-id>
     Applies the proposal. Requires --yes and --dry-run=false flags.

WORKFLOW EXAMPLE:

  # Step 1: Propose a change
  $ echo "version = 2.0" | goshi fs write VERSION.txt
  Proposal ID: abc123def456

  # Step 2: Review what would happen
  $ goshi fs apply abc123def456 --dry-run
  [dry-run] would write: VERSION.txt

  # Step 3: Apply the change
  $ goshi fs apply abc123def456 --dry-run=false --yes
  ✔ file written successfully

SEE ALSO:
  goshi help fs read    - Read a file
  goshi help fs list    - List a directory
  goshi help fs write   - Write a file
  goshi help fs apply   - Apply a write proposal`,
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
  }`,
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
