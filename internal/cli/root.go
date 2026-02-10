package cli

import (
	"fmt"
	"os"

	"github.com/cshaiku/goshi/internal/config"
	"github.com/spf13/cobra"
)

var runtime *Runtime

// globalConfig holds the loaded config for access throughout CLI
var globalConfig *config.Config

// Mode flags
var (
	headlessMode bool
)

var rootCmd = &cobra.Command{
	Use:   "goshi",
	Short: "Goshi is a local-first protective CLI agent",
	Long: `Goshi is a local-first protective CLI agent for safe and auditable filesystem operations.

OVERVIEW:
Goshi helps you diagnose environment issues, repair problems safely, and perform
filesystem operations with full auditability and reversibility. All operations are
scoped to your repository root and require explicit user confirmation.

QUICK START:

  1. Check your environment:
     $ goshi doctor
     Diagnose environment health and identify any issues.

  2. View or manage configuration:
     $ goshi config show
     Display current configuration.
     
     $ goshi config init
     Generate default configuration template.

  3. Work with files safely:
     $ goshi fs read src/main.go
     Read file contents safely.
     
     $ goshi fs list .
     List directory contents.
     
     $ echo "content" | goshi fs write file.txt
     Propose a file write.

  4. Repair issues automatically:
     $ goshi heal --dry-run=false --yes
     Automatically repair detected environment issues.

DOCUMENTATION:
Each command provides detailed help with --help:
  $ goshi [command] --help

For specific topics:
  $ goshi help config   - Configuration management
  $ goshi help fs       - Filesystem operations
  $ goshi help doctor   - Environment diagnostics
  $ goshi help heal     - Automatic repairs

INTERACTIVE MODE:
Run goshi without arguments to start an interactive chat with the AI agent:
  $ goshi

ENVIRONMENT VARIABLES:
  GOSHI_CONFIG        - Path to configuration file (overrides file search)
  GOSHI_MODEL         - LLM model to use (overrides config file)
  GOSHI_LLM_PROVIDER  - LLM provider: ollama, openai, etc.
  GOSHI_OLLAMA_URL    - Ollama server URL
  GOSHI_OLLAMA_PORT   - Ollama server port number`,

	Run: func(cmd *cobra.Command, args []string) {
		// If any args are present, Cobra is resolving a subcommand.
		// Root Run must NOT execute.
		if len(args) > 0 {
			return
		}

		if runtime == nil || runtime.SystemPrompt == nil {
			fmt.Fprintln(os.Stderr, "fatal: system prompt not initialized")
			os.Exit(1)
		}

		// Check if we should run in TUI or headless/CLI mode
		if headlessMode {
			runChat(runtime.SystemPrompt.Raw())
		} else {
			runTUIMode(runtime.SystemPrompt.Raw())
		}
	},
}

// GetConfig returns the globally loaded config
func GetConfig() *config.Config {
	if globalConfig == nil {
		cfg := config.Load()
		globalConfig = &cfg
	}
	return globalConfig
}

func Execute(rt *Runtime) {
	runtime = rt
	cfg := config.Load()
	globalConfig = &cfg

	// Add mode flags
	rootCmd.PersistentFlags().BoolVar(&headlessMode, "headless", false, "Run in headless/CLI mode (no TUI)")

	// Register all subcommands
	rootCmd.AddCommand(
		newFSCommand(),
		newDoctorCmd(&cfg),
		newHealCmd(&cfg),
		newConfigCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
