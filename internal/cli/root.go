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

var rootCmd = &cobra.Command{
	Use:   "goshi",
	Short: "Goshi is a local-first protective CLI agent",

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

		runChat(runtime.SystemPrompt.Raw())
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
