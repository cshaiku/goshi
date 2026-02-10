package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/cshaiku/goshi/internal/experiments"
	"github.com/cshaiku/goshi/internal/llm"
	"github.com/cshaiku/goshi/internal/llm/ollama"
	"github.com/spf13/cobra"
)

func newFSProbeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fs-probe",
		Short: "Run filesystem handshake probe",
		Long: `Experimental probe for filesystem capability handshake with LLM.

DESCRIPTION:
This is an experimental command that tests the LLM's ability to interact with
the filesystem through a structured handshake protocol. It explores how well
the AI can understand and request specific files without hallucination.

PURPOSE:
The filesystem handshake probe validates that:
  - LLM recognizes file access is constrained
  - LLM can properly request specific files
  - LLM understands filesystem structure
  - LLM doesn't guess or hallucinate file contents

WORKFLOW:
  1. Lists files in current directory
  2. Provides filenames to LLM (without contents)
  3. LLM analyzes and requests which files to read
  4. Validates that requested files exist and match intent
  5. Reports results of the handshake

OUTPUT:
Produces diagnostic output showing:
  - Files discovered
  - LLM requests made
  - File access patterns
  - Handshake success/failure

USAGE:
  $ goshi fs-probe
  Runs probe in current working directory.

NOTES:
- This is an experimental feature and may change
- Requires LLM backend to be running (Ollama by default)
- Useful for testing LLM safety constraints
- Not recommended for production workflows

ENVIRONMENT:
  GOSHI_MODEL         - LLM model to use (default: ollama)
  GOSHI_LLM_PROVIDER  - LLM provider (default: auto)

SEE ALSO:
  goshi fs            - Main filesystem operations (read, write, list, apply)
  goshi help          - Show general help information`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime == nil || runtime.SystemPrompt == nil {
				return fmt.Errorf("runtime not initialized")
			}

			cfg := config.Load()
			ctx := context.Background()

			var backend llm.Backend
			if cfg.LLMProvider == "" || cfg.LLMProvider == "auto" || cfg.LLMProvider == "ollama" {
				backend = ollama.New(cfg.Model)
			} else {
				return fmt.Errorf("unsupported LLM provider")
			}

			sp, err := llm.NewSystemPrompt(runtime.SystemPrompt.Raw())
			if err != nil {
				return err
			}

			client := llm.NewClient(sp, backend)

			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			cwd, err = filepath.EvalSymlinks(cwd)
			if err != nil {
				return err
			}

			return experiments.RunFSHandshakeProbe(ctx, client, cwd)
		},
	}
}
